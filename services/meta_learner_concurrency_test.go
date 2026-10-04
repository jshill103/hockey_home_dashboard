package services

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

// syntheticMetaTrainingSet builds a separable training set large enough to
// clear Train's 20-example minimum and reach its validation epochs, which is
// where the self-deadlock used to trigger.
func syntheticMetaTrainingSet(n int) []MetaTrainingExample {
	rng := rand.New(rand.NewSource(1))
	data := make([]MetaTrainingExample, 0, n)

	for i := 0; i < n; i++ {
		homeWin := i%2 == 0

		// Base models agree with the outcome, with noise, so the meta-learner
		// has a real signal to fit rather than pure noise.
		base := 0.35
		if homeWin {
			base = 0.65
		}
		jitter := func() float64 { return base + (rng.Float64()-0.5)*0.1 }

		outcome := 0.0
		if homeWin {
			outcome = 1.0
		}

		data = append(data, MetaTrainingExample{
			Predictions: ModelPredictions{
				Statistical:      jitter(),
				Bayesian:         jitter(),
				MonteCarlo:       jitter(),
				Elo:              jitter(),
				Poisson:          jitter(),
				NeuralNetwork:    jitter(),
				GradientBoosting: jitter(),
				LSTM:             jitter(),
				RandomForest:     jitter(),
			},
			Context: MetaGameContext{
				IsDivisionalGame: i%3 == 0,
				HomeTeamHot:      homeWin,
				RestAdvantage:    float64(i%4) - 1.5,
				TravelDistance:   float64(i%7) * 100,
			},
			ActualOutcome: outcome,
			GameID:        1000 + i,
			GameDate:      time.Now().AddDate(0, 0, -n+i),
		})
	}

	return data
}

// TestTrainDoesNotDeadlockOnItself pins the bug where Train held the write
// lock for its whole body and reached PredictFromModels (via evaluateAccuracy),
// which took RLock on the same mutex. sync.RWMutex is not reentrant, so Train
// blocked forever on its first validation epoch and never released the write
// lock, wedging every subsequent reader.
func TestTrainDoesNotDeadlockOnItself(t *testing.T) {
	mlm := NewMetaLearnerModel()

	done := make(chan error, 1)
	go func() { done <- mlm.Train(syntheticMetaTrainingSet(200)) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Train returned an error: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Train did not return within 30s; it deadlocked against its own write lock")
	}
}

// TestIsTrainedDoesNotBlockBehindTraining is the symptom the deadlock produced
// in production: prediction requests calling IsTrained() parked on RLock
// behind the wedged trainer and the HTTP handler never returned.
func TestIsTrainedDoesNotBlockBehindTraining(t *testing.T) {
	mlm := NewMetaLearnerModel()

	var wg sync.WaitGroup
	stop := make(chan struct{})
	reads := make(chan int, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		count := 0
		for {
			select {
			case <-stop:
				reads <- count
				return
			default:
				_ = mlm.IsTrained()
				count++
			}
		}
	}()

	trained := make(chan error, 1)
	go func() { trained <- mlm.Train(syntheticMetaTrainingSet(200)) }()

	select {
	case err := <-trained:
		if err != nil {
			t.Fatalf("Train returned an error: %v", err)
		}
	case <-time.After(30 * time.Second):
		close(stop)
		wg.Wait()
		t.Fatal("Train did not finish within 30s")
	}

	close(stop)
	wg.Wait()

	if n := <-reads; n == 0 {
		t.Fatal("IsTrained never completed a read while training ran")
	}

	if !mlm.IsTrained() {
		t.Fatal("meta-learner should report trained after a successful Train")
	}
}

// TestPredictFromModelsMatchesLockedVariant guards the refactor itself: the
// public method and the lock-free variant must agree, so splitting them did
// not change behaviour.
func TestPredictFromModelsMatchesLockedVariant(t *testing.T) {
	mlm := NewMetaLearnerModel()
	if err := mlm.Train(syntheticMetaTrainingSet(200)); err != nil {
		t.Fatalf("Train returned an error: %v", err)
	}

	for _, example := range syntheticMetaTrainingSet(10) {
		public := mlm.PredictFromModels(&example.Predictions, &example.Context)

		mlm.mutex.RLock()
		locked := mlm.predictFromModelsLocked(&example.Predictions, &example.Context)
		mlm.mutex.RUnlock()

		if public != locked {
			t.Fatalf("PredictFromModels=%v but predictFromModelsLocked=%v", public, locked)
		}
	}
}
