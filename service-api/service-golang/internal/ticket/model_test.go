package ticket

import "testing"

// TestUpsertInputValidate garante que um chamado válido seja aceito.
func TestUpsertInputValidate(t *testing.T) {
	t.Parallel()

	valid := UpsertInput{
		Title:          "Notebook sem acesso à rede",
		Description:    "O equipamento não conecta à rede interna.",
		RequesterName:  "Maria",
		Priority:       PriorityHigh,
		Status:         StatusOpen,
		AssignmentMode: AssignmentAutomatic,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
}

// TestUpsertInputValidateRequiresAssigneeForManualAssignment garante a seleção manual obrigatória.
func TestUpsertInputValidateRequiresAssigneeForManualAssignment(t *testing.T) {
	t.Parallel()

	input := UpsertInput{
		Title:          "Troca de cadeira",
		Description:    "Encosto quebrado.",
		RequesterName:  "João",
		Priority:       PriorityMedium,
		Status:         StatusOpen,
		AssignmentMode: AssignmentManual,
	}

	err := input.Validate()
	problems, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if problems["assigneeId"] == "" {
		t.Fatal("expected assigneeId validation error")
	}
}

// TestUpsertInputNormalize garante a remoção de espaços excedentes nos campos textuais.
func TestUpsertInputNormalize(t *testing.T) {
	t.Parallel()

	input := UpsertInput{Title: "  Impressora  ", RequesterName: " Ana "}
	input.Normalize()

	if input.Title != "Impressora" || input.RequesterName != "Ana" {
		t.Fatalf("input was not normalized: %#v", input)
	}
}

// TestAssigneeInputValidate garante que membros sem nome sejam rejeitados.
func TestAssigneeInputValidate(t *testing.T) {
	t.Parallel()

	input := AssigneeInput{Name: "   "}
	input.Normalize()

	err := input.Validate()
	problems, ok := err.(ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if problems["name"] == "" {
		t.Fatal("expected name validation error")
	}
}
