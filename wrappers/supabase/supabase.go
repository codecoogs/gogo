package codecoogssupabase

import (
	"errors"
	"github.com/supabase-community/supabase-go"
	"os"
)

func CreateClient() (*supabase.Client, error) {
	supabaseUrl := os.Getenv("SUPABASE_URL")
	if supabaseUrl == "" {
		return nil, errors.New("SUPABASE_URL is required.")
	}

	supabaseKey := os.Getenv("SUPABASE_KEY")
	if supabaseKey == "" {
		return nil, errors.New("SUPABASE_KEY is required.")
	}

	client, err := supabase.NewClient(supabaseUrl, supabaseKey, nil)

	return client, err
}