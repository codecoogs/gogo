package codecoogssupabase

import (
	"errors"
	"os"

	"github.com/supabase-community/supabase-go"
)

func CreateClient() (*supabase.Client, error) {
	supabaseUrl := os.Getenv("SUPABASE_URL")
	if supabaseUrl == "" {
		return nil, errors.New("SUPABASE_URL is required")
	}

	supabaseKey := os.Getenv("SUPABASE_KEY")
	if supabaseKey == "" {
		return nil, errors.New("SUPABASE_KEY is required")
	}

	// fmt.Println(supabaseUrl)
	// fmt.Println(supabaseKey)

	client, err := supabase.NewClient("https://hbhahqephsndzjtqdzin.supabase.co", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImhiaGFocWVwaHNuZHpqdHFkemluIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImlhdCI6MTcwMDUyNjI1OCwiZXhwIjoyMDE2MTAyMjU4fQ._UGTH-ZWQ5Ho0SoUBf81TR93T6iYcHG3tBLaxnQiReA", nil)

	return client, err
}
