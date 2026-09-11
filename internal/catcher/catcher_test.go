package catcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	homerun "github.com/stuttgart-things/homerun-library/v4"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/dashboard"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/mock"
	"github.com/stuttgart-things/homerun2-light-catcher/internal/models"
)

const (
	testStream = "tabletennis"
	testGroup  = "homerun2-light-catcher"
)

// redisStackHook fills the gaps between miniredis and the Redis Stack the
// catcher runs against: redisqueue's preflight reads the server version from
// INFO, and payloads are resolved with JSON.GET. JSON documents are stored as
// plain string keys.
type redisStackHook struct {
	mr *miniredis.Miniredis
}

func (redisStackHook) DialHook(next redis.DialHook) redis.DialHook { return next }

func (h redisStackHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		switch cmd.Name() {
		case "info":
			cmd.(*redis.StringCmd).SetVal("# Server\r\nredis_version:7.2.0\r\n")
			return nil
		case "json.get":
			key := fmt.Sprint(cmd.Args()[1])
			val, err := h.mr.Get(key)
			if err != nil {
				cmd.SetErr(redis.Nil)
				return redis.Nil
			}
			cmd.(*redis.Cmd).SetVal(val)
			return nil
		}
		return next(ctx, cmd)
	}
}

func (redisStackHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

type fakeRedis struct {
	mr     *miniredis.Miniredis
	client *redis.Client
}

func newFakeRedis(t *testing.T) *fakeRedis {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	client.AddHook(redisStackHook{mr: mr})
	return &fakeRedis{mr: mr, client: client}
}

// pitch stores the message payload and appends its ID to the stream, the way
// homerun2 pitchers do.
func (f *fakeRedis) pitch(t *testing.T, id string, msg homerun.Message) {
	t.Helper()
	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.mr.Set(id, string(body)); err != nil {
		t.Fatal(err)
	}
	if err := f.client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: testStream,
		Values: map[string]any{"messageID": id},
	}).Err(); err != nil {
		t.Fatal(err)
	}
}

// startCatcher runs a catcher against the fake Redis and waits until its
// consumer group exists. It stops the catcher when the test ends.
func (f *fakeRedis) startCatcher(t *testing.T, startID string, handlers ...MessageHandler) {
	t.Helper()

	orig := blockingTimeout
	blockingTimeout = 50 * time.Millisecond
	t.Cleanup(func() { blockingTimeout = orig })

	c, err := newRedisCatcher(f.client, f.client, "", []string{testStream}, testGroup, "test", startID, handlers...)
	if err != nil {
		t.Fatalf("newRedisCatcher: %v", err)
	}

	go func() {
		for range c.Errors() {
		}
	}()

	done := make(chan struct{})
	go func() {
		c.Run()
		close(done)
	}()
	t.Cleanup(func() {
		c.Shutdown()
		<-done
	})

	deadline := time.Now().Add(2 * time.Second)
	for {
		groups, _ := f.client.XInfoGroups(context.Background(), testStream).Result()
		if len(groups) > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("consumer group was never created")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// recorder is a MessageHandler that records the object IDs it handled.
type recorder struct {
	mu  sync.Mutex
	ids []string
}

func (r *recorder) handle(msg models.CaughtMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ids = append(r.ids, msg.ObjectID)
}

// waitFor waits until n messages were handled and returns their IDs.
func (r *recorder) waitFor(t *testing.T, n int) []string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		r.mu.Lock()
		ids := slices.Clone(r.ids)
		r.mu.Unlock()
		if len(ids) >= n {
			return ids
		}
		if time.Now().After(deadline) {
			t.Fatalf("handled %d of %d messages: %v", len(ids), n, ids)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestRedisCatcher_NewGroupSkipsBacklog(t *testing.T) {
	f := newFakeRedis(t)
	f.pitch(t, "backlog-1", homerun.Message{System: testStream})
	f.pitch(t, "backlog-2", homerun.Message{System: testStream})

	rec := &recorder{}
	f.startCatcher(t, "", rec.handle)

	f.pitch(t, "live-1", homerun.Message{System: testStream})

	// Messages are handled in stream order, so a replayed backlog would show
	// up before live-1.
	if ids := rec.waitFor(t, 1); !slices.Equal(ids, []string{"live-1"}) {
		t.Fatalf("handled %v, want only the message pitched after start", ids)
	}
}

func TestRedisCatcher_ExistingGroupKeepsPosition(t *testing.T) {
	f := newFakeRedis(t)
	f.pitch(t, "handled-by-previous-run", homerun.Message{System: testStream})

	// An earlier run created the group; then the catcher was down while
	// another message was pitched.
	if err := f.client.XGroupCreateMkStream(context.Background(), testStream, testGroup, "$").Err(); err != nil {
		t.Fatal(err)
	}
	f.pitch(t, "pitched-while-down", homerun.Message{System: testStream})

	rec := &recorder{}
	f.startCatcher(t, "", rec.handle)

	if ids := rec.waitFor(t, 1); !slices.Equal(ids, []string{"pitched-while-down"}) {
		t.Fatalf("handled %v, want the message pitched while the catcher was down", ids)
	}
}

func TestRedisCatcher_StartIDZeroReplaysStream(t *testing.T) {
	f := newFakeRedis(t)
	f.pitch(t, "backlog-1", homerun.Message{System: testStream})

	rec := &recorder{}
	f.startCatcher(t, "0", rec.handle)

	if ids := rec.waitFor(t, 1); !slices.Equal(ids, []string{"backlog-1"}) {
		t.Fatalf("handled %v, want the backlog replayed with start ID 0", ids)
	}
}

func TestNewRedisCatcher_InvalidStartID(t *testing.T) {
	for _, id := range []string{"latest", ">", "-1", "1-", "$0"} {
		f := newFakeRedis(t)
		if _, err := newRedisCatcher(f.client, f.client, "", []string{testStream}, testGroup, "test", id); err == nil {
			t.Errorf("start ID %q: expected error", id)
		}
	}
}

func TestRedisCatcher_HandlesStreamInOrder(t *testing.T) {
	f := newFakeRedis(t)

	rec := &recorder{}
	slowOnEven := func(msg models.CaughtMessage) {
		// Uneven handling time lets any concurrent worker overtake another.
		if n, _ := strconv.Atoi(msg.Title); n%2 == 0 {
			time.Sleep(2 * time.Millisecond)
		}
		rec.handle(msg)
	}
	f.startCatcher(t, "", slowOnEven)

	var want []string
	for i := range 40 {
		id := fmt.Sprintf("msg-%02d", i)
		want = append(want, id)
		f.pitch(t, id, homerun.Message{System: testStream, Title: strconv.Itoa(i)})
	}

	if ids := rec.waitFor(t, len(want)); !slices.Equal(ids, want) {
		t.Fatalf("handled out of stream order:\n got %v\nwant %v", ids, want)
	}
}

// TestRedisCatcher_BurstEndsOnLastEffect replays the button hub case from #55:
// a point and the match-winning point pitched 1 ms apart. The match effect
// must be the one left on, and the point's short timer must not turn it off.
func TestRedisCatcher_BurstEndsOnLastEffect(t *testing.T) {
	srv := httptest.NewServer(mock.NewServer("test", "abc1234", "2026-01-01").Handler())
	defer srv.Close()

	profileYAML := fmt.Sprintf(`effects:
  tabletennis-match:
    systems: [tabletennis]
    severity: [success]
    tags: [transition=match_won]
    fx: Fireworks
    duration: 10
    color: forest
    endpoint: %[1]s
  tabletennis-point:
    systems: [tabletennis]
    severity: [info]
    tags: [transition=point]
    fx: Solid
    duration: 1
    color: blue
    endpoint: %[1]s
`, srv.URL)
	profilePath := filepath.Join(t.TempDir(), "profile.yaml")
	if err := os.WriteFile(profilePath, []byte(profileYAML), 0644); err != nil {
		t.Fatal(err)
	}

	f := newFakeRedis(t)
	tracker := dashboard.NewEventTracker()
	// Handling the point takes a little longer (a slower payload lookup or
	// log write), which is all a concurrent worker needs to let the match
	// overtake it.
	slowPoint := func(msg models.CaughtMessage) {
		if msg.ObjectID == "point" {
			time.Sleep(20 * time.Millisecond)
		}
	}
	f.startCatcher(t, "", slowPoint, LightHandler(profilePath, tracker))

	now := strconv.FormatInt(time.Now().Unix(), 10)
	f.pitch(t, "point", homerun.Message{System: "tabletennis", Severity: "info", Tags: "set=3,transition=point,side=a", Timestamp: now})
	time.Sleep(time.Millisecond)
	f.pitch(t, "match", homerun.Message{System: "tabletennis", Severity: "success", Tags: "set=3,transition=match_won,side=a", Timestamp: now})

	deadline := time.Now().Add(3 * time.Second)
	for tracker.Count() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("triggered %d of 2 effects", tracker.Count())
		}
		time.Sleep(10 * time.Millisecond)
	}

	events := tracker.Events()
	if events[0].Effect != "Solid" || events[1].Effect != "Fireworks" {
		t.Fatalf("effects triggered as %s, %s; want Solid, Fireworks", events[0].Effect, events[1].Effect)
	}

	// Past the point's one-second duration.
	time.Sleep(1300 * time.Millisecond)

	resp, err := http.Get(srv.URL + "/json/state")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var state mock.WLEDState
	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		t.Fatal(err)
	}
	if !state.On || len(state.Seg) == 0 || state.Seg[0].Fx != 6 {
		t.Fatalf("light should still show Fireworks (fx 6), got %+v", state)
	}
}
