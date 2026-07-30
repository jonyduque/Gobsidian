package writer_test

import (
	"sync"
	"testing"
	"time"

	"github.com/jonyd/gobsidian/internal/vault"
	"github.com/jonyd/gobsidian/internal/writer"
)

func TestPathLocker_SamePathLostUpdate(t *testing.T) {
	locker := writer.NewPathLocker()
	p := vault.CanonicalPath("nota.md")

	var counter int
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock := locker.Lock(p)
			defer unlock()

			// Simula secao critica de leitura + modificacao + escrita
			val := counter
			time.Sleep(1 * time.Millisecond)
			counter = val + 1
		}()
	}

	wg.Wait()

	if counter != 50 {
		t.Fatalf("counter = %d, quer 50 (lost update ocorreu)", counter)
	}
}

func TestPathLocker_SamePathCasing(t *testing.T) {
	locker := writer.NewPathLocker()
	p1 := vault.CanonicalPath("Civil/A.md")
	p2 := vault.CanonicalPath("civil/a.md")

	unlock1 := locker.Lock(p1)

	acquired2 := make(chan bool)
	go func() {
		unlock2 := locker.Lock(p2)
		acquired2 <- true
		unlock2()
	}()

	select {
	case <-acquired2:
		t.Fatal("p2 (civil/a.md) adquiriu a trava enquanto p1 (Civil/A.md) ainda estava travado — casing insensivel falhou")
	case <-time.After(50 * time.Millisecond):
		// Sucesso: p2 ficou bloqueado aguardando p1
	}

	unlock1()

	select {
	case <-acquired2:
		// Sucesso: p2 adquiriu apos p1 destravar
	case <-time.After(1 * time.Second):
		t.Fatal("p2 nao adquiriu a trava apos p1 destravar")
	}
}

func TestPathLocker_DifferentPathsParallel(t *testing.T) {
	locker := writer.NewPathLocker()
	p1 := vault.CanonicalPath("note1.md")
	p2 := vault.CanonicalPath("note2.md")

	unlock1 := locker.Lock(p1)
	defer unlock1()

	start := time.Now()
	acquired2 := make(chan bool, 1)

	go func() {
		unlock2 := locker.Lock(p2)
		acquired2 <- true
		unlock2()
	}()

	select {
	case <-acquired2:
		if dur := time.Since(start); dur > 100*time.Millisecond {
			t.Fatalf("p2 demorou %v para adquirir a trava; esperava execucao paralela imediata", dur)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("p2 nao adquiriu a trava em caminho diferente (trava global indevida)")
	}
}

func TestPathLocker_NoMemoryLeak(t *testing.T) {
	locker := writer.NewPathLocker()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		path := vault.CanonicalPath(t.TempDir() + "/note.md")
		go func(p vault.CanonicalPath) {
			defer wg.Done()
			unlock := locker.Lock(p)
			time.Sleep(1 * time.Millisecond)
			unlock()
		}(path)
	}

	wg.Wait()

	if count := locker.Count(); count != 0 {
		t.Fatalf("sobram %d travas no registro; esperava 0 (vazamento de memoria)", count)
	}
}
