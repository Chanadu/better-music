// auth-admin generates admin recovery links or retries pending account changes.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Chanadu/better-music/config"
	"github.com/Chanadu/better-music/db"
	"github.com/Chanadu/better-music/internal/auth"
)

func main() {
	user := flag.String("user", "", "SuperTokens account ID")
	resetLink := flag.Bool("reset-link", false, "print a private single-use reset link for -user ID")
	reconcile := flag.Bool("reconcile", false, "retry pending email/deletion operations")
	flag.Parse()

	switch {
	case *user != "" && *resetLink && !*reconcile:
	case *reconcile && *user == "" && !*resetLink:
	default:
		log.Fatal("choose -user ID -reset-link or -reconcile")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	database := db.Connect(cfg.DB.Url)
	defer database.Close()
	if err := db.RequireAuthSchema(database); err != nil {
		log.Fatal(err)
	}
	service, err := auth.Init(cfg.Auth, database)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	if *reconcile {
		if err := service.ReconcilePending(ctx); err != nil {
			log.Fatal(err)
		}
		return
	}

	link, err := service.PasswordResetLink(ctx, *user)
	if err != nil {
		log.Fatalf("could not create reset link for account %s: %v", *user, err)
	}
	fmt.Fprintln(os.Stdout, link)
}
