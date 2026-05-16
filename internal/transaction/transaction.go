package transaction

import (
	"fmt"
	"log"
	"os"

	"clashtui/internal/backup"
)

type Transaction struct {
	name       string
	prepareFn  func() (backupPath string, err error)
	commitFn   func() error
	verifyFn   func() error
	rollbackFn func(backupPath string) error
	backupPath string
	logger     *log.Logger
}

func NewTransaction(name string) *Transaction {
	return &Transaction{
		name:   name,
		logger: log.New(os.Stderr, "[transaction] ", log.LstdFlags),
	}
}

func (t *Transaction) Prepare(prepareFn func() (backupPath string, err error)) *Transaction {
	t.prepareFn = prepareFn
	return t
}

func (t *Transaction) Commit(commitFn func() error) *Transaction {
	t.commitFn = commitFn
	return t
}

func (t *Transaction) Verify(verifyFn func() error) *Transaction {
	t.verifyFn = verifyFn
	return t
}

func (t *Transaction) Rollback(rollbackFn func(backupPath string) error) *Transaction {
	t.rollbackFn = rollbackFn
	return t
}

func (t *Transaction) Execute() error {
	t.logger.Printf("Starting transaction: %s", t.name)

	if t.prepareFn != nil {
		t.logger.Printf("Prepare phase: %s", t.name)
		backupPath, err := t.prepareFn()
		if err != nil {
			t.logger.Printf("Prepare failed: %v", err)
			return fmt.Errorf("prepare failed: %w", err)
		}
		t.backupPath = backupPath
		t.logger.Printf("Prepare complete, backup at: %s", backupPath)
	}

	if t.commitFn != nil {
		t.logger.Printf("Commit phase: %s", t.name)
		if err := t.commitFn(); err != nil {
			t.logger.Printf("Commit failed: %v", err)
			if t.rollbackFn != nil && t.backupPath != "" {
				t.logger.Printf("Rolling back: %s", t.name)
				if rbErr := t.rollbackFn(t.backupPath); rbErr != nil {
					t.logger.Printf("Rollback also failed: %v", rbErr)
					return fmt.Errorf("commit failed: %w, rollback also failed: %v", err, rbErr)
				}
				t.logger.Printf("Rollback complete: %s", t.name)
			}
			return fmt.Errorf("commit failed: %w", err)
		}
		t.logger.Printf("Commit complete: %s", t.name)
	}

	if t.verifyFn != nil {
		t.logger.Printf("Verify phase: %s", t.name)
		if err := t.verifyFn(); err != nil {
			t.logger.Printf("Verify failed: %v", err)
			if t.rollbackFn != nil && t.backupPath != "" {
				t.logger.Printf("Rolling back: %s", t.name)
				if rbErr := t.rollbackFn(t.backupPath); rbErr != nil {
					t.logger.Printf("Rollback also failed: %v", rbErr)
					return fmt.Errorf("verify failed: %w, rollback also failed: %v", err, rbErr)
				}
				t.logger.Printf("Rollback complete: %s", t.name)
			}
			return fmt.Errorf("verify failed: %w", err)
		}
		t.logger.Printf("Verify complete: %s", t.name)
	}

	t.logger.Printf("✓ Transaction complete: %s", t.name)
	return nil
}

func ConfigTransaction(configPath string, operation string, commitFn func() error) error {
	return NewTransaction(operation).
		Prepare(func() (string, error) {
			return backup.CreateBackup(configPath)
		}).
		Commit(commitFn).
		Verify(func() error {
			_, err := os.ReadFile(configPath)
			if err != nil {
				return err
			}
			return nil
		}).
		Rollback(func(backupPath string) error {
			return backup.RestoreBackup(configPath, backupPath)
		}).
		Execute()
}
