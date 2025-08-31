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

	client, err := supabase.NewClient("https://hbhahqephsndzjtqdzin.supabase.co", supabaseKey, nil)

	return client, err
}
