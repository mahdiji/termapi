package requestinput

import "testing"

func TestFindMatchesAcrossLines(t *testing.T) {
	p := NewResponsePanel()
	p.SetSize(90, 18)
	p.SetResponse(ResponseMsg{
		Status:  "200 OK",
		Code:    200,
		Body:    "alpha\nbeta\nbeta two\ngamma",
		RawBody: "alpha\nbeta\nbeta two\ngamma",
	})

	p.Find.SetValue("BETA")
	p.recomputeMatches()
	if len(p.Matches) != 2 || p.Matches[0] != 1 || p.Matches[1] != 2 {
		t.Fatalf("matches = %v, want [1 2]", p.Matches)
	}

	p.NextMatch(1)
	if p.MatchIdx != 1 {
		t.Fatalf("match index = %d, want 1", p.MatchIdx)
	}
	p.NextMatch(1)
	if p.MatchIdx != 0 {
		t.Fatalf("match index should wrap to 0, got %d", p.MatchIdx)
	}
	p.NextMatch(-1)
	if p.MatchIdx != 1 {
		t.Fatalf("previous match should go back to 1, got %d", p.MatchIdx)
	}

	p.CloseFind()
	if p.FindActive() || len(p.Matches) != 0 {
		t.Fatal("close find should clear the query and matches")
	}
}

func TestFindNoMatches(t *testing.T) {
	p := NewResponsePanel()
	p.SetSize(90, 18)
	p.SetResponse(ResponseMsg{Status: "200 OK", Code: 200, Body: "alpha\nbeta", RawBody: "alpha\nbeta"})

	p.Find.SetValue("zzz")
	p.recomputeMatches()
	if len(p.Matches) != 0 {
		t.Fatalf("matches = %v, want none", p.Matches)
	}
	p.NextMatch(1)
	if p.MatchIdx != 0 {
		t.Fatalf("match index = %d, want 0", p.MatchIdx)
	}
}
