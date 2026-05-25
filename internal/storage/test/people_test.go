package storage_test

import (
	"testing"

	"github.com/mohitraku/sica/internal/models"
	"github.com/mohitraku/sica/internal/storage"
)

func samplePerson(id, name string) *models.Person {
	now := models.NowUTC()
	return &models.Person{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestPersonCreate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewPersonStore(db)

	p := samplePerson("a1b2c3d4e5f6a7b8", "John Doe")
	p.Email = "john@example.com"
	p.Phone = "555-0100"
	if err := store.Create(p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.GetByID("a1b2c3d4e5f6a7b8")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil for existing person")
	}
	if got.Name != "John Doe" {
		t.Errorf("expected name 'John Doe', got %q", got.Name)
	}
	if got.Email != "john@example.com" {
		t.Errorf("expected email 'john@example.com', got %q", got.Email)
	}
	if got.Phone != "555-0100" {
		t.Errorf("expected phone '555-0100', got %q", got.Phone)
	}
}

func TestPersonGetByIDNotFound(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewPersonStore(db)

	got, err := store.GetByID("nonexistent")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent person")
	}
}

func TestPersonList(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewPersonStore(db)

	p1 := samplePerson("id1", "Alice")
	p1.LastContacted = "2026-05-20"
	if err := store.Create(p1); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	p2 := samplePerson("id2", "Bob")
	p2.LastContacted = "2026-05-25"
	if err := store.Create(p2); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	p3 := samplePerson("id3", "Charlie")
	if err := store.Create(p3); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	people, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(people) != 3 {
		t.Errorf("expected 3 people, got %d", len(people))
	}
	// Never-contacted first
	if people[0].Name != "Charlie" || people[0].LastContacted != "" {
		t.Errorf("expected first person to be never-contacted 'Charlie', got %q (last=%q)", people[0].Name, people[0].LastContacted)
	}
	// Then longest-ago
	if people[1].Name != "Alice" {
		t.Errorf("expected second person to be 'Alice', got %q", people[1].Name)
	}
	if people[2].Name != "Bob" {
		t.Errorf("expected third person to be 'Bob', got %q", people[2].Name)
	}
}

func TestPersonListEmpty(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewPersonStore(db)

	people, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(people) != 0 {
		t.Errorf("expected 0 people, got %d", len(people))
	}
}

func TestPersonUpdate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewPersonStore(db)

	p := samplePerson("id1", "Original")
	if err := store.Create(p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	p.Name = "Updated"
	p.Email = "new@example.com"
	p.Phone = "555-9999"
	p.LastContacted = "2026-05-25"
	p.UpdatedAt = models.NowUTC()
	if err := store.Update(p); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := store.GetByID("id1")
	if got.Name != "Updated" {
		t.Errorf("expected 'Updated', got %q", got.Name)
	}
	if got.Email != "new@example.com" {
		t.Errorf("expected 'new@example.com', got %q", got.Email)
	}
	if got.LastContacted != "2026-05-25" {
		t.Errorf("expected last_contacted '2026-05-25', got %q", got.LastContacted)
	}
}

func TestPersonUpdateNonexistent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewPersonStore(db)

	p := samplePerson("nope", "Ghost")
	err := store.Update(p)
	if err == nil {
		t.Error("expected error updating nonexistent person")
	}
}

func TestPersonDelete(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewPersonStore(db)

	if err := store.Create(samplePerson("id1", "ToDelete")); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.Delete("id1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := store.GetByID("id1")
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestPersonDeleteNonexistent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewPersonStore(db)

	err := store.Delete("nope")
	if err == nil {
		t.Error("expected error deleting nonexistent person")
	}
}

func TestSignificantDatesCRUD(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewPersonStore(db)

	p := samplePerson("id1", "John")
	if err := store.Create(p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	dates := []models.SignificantDate{
		{Label: "Birthday", Date: "05-03"},
		{Label: "Anniversary", Date: "09-15"},
	}
	if err := store.SetSignificantDates("id1", dates); err != nil {
		t.Fatalf("SetSignificantDates failed: %v", err)
	}

	got, err := store.GetSignificantDates("id1")
	if err != nil {
		t.Fatalf("GetSignificantDates failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 dates, got %d", len(got))
	}
	if got[0].Label != "Birthday" || got[0].Date != "05-03" {
		t.Errorf("expected first date sorted 'Birthday' (05-03), got %q (%s)", got[0].Label, got[0].Date)
	}
	if got[1].Label != "Anniversary" || got[1].Date != "09-15" {
		t.Errorf("expected second date sorted 'Anniversary' (09-15), got %q (%s)", got[1].Label, got[1].Date)
	}

	// Replace dates
	newDates := []models.SignificantDate{
		{Label: "Birthday", Date: "06-01"},
	}
	if err := store.SetSignificantDates("id1", newDates); err != nil {
		t.Fatalf("SetSignificantDates (replace) failed: %v", err)
	}
	got, _ = store.GetSignificantDates("id1")
	if len(got) != 1 {
		t.Fatalf("expected 1 date after replace, got %d", len(got))
	}
	if got[0].Label != "Birthday" || got[0].Date != "06-01" {
		t.Errorf("expected 'Birthday' (06-01), got %q (%s)", got[0].Label, got[0].Date)
	}
}

func TestSignificantDatesCascadeDelete(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	store := storage.NewPersonStore(db)

	p := samplePerson("id1", "John")
	if err := store.Create(p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	dates := []models.SignificantDate{
		{Label: "Birthday", Date: "05-03"},
	}
	if err := store.SetSignificantDates("id1", dates); err != nil {
		t.Fatalf("SetSignificantDates failed: %v", err)
	}

	if err := store.Delete("id1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := store.GetSignificantDates("id1")
	if len(got) != 0 {
		t.Errorf("expected 0 dates after cascade delete, got %d", len(got))
	}
}
