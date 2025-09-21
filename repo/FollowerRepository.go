package repo

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type FollowerRepository struct {
	driver neo4j.DriverWithContext
	logger *log.Logger
}

func NewFollowerRepository(logger *log.Logger) (*FollowerRepository, error) {
	// Podrži oba seta imena env varijabli (kao u Stakeholders i kao u ranijem compose-u)
	uri := firstNonEmpty(os.Getenv("NEO4J_DB"), os.Getenv("NEO4J_URI"))
	user := firstNonEmpty(os.Getenv("NEO4J_USERNAME"), os.Getenv("NEO4J_USER"))
	pass := os.Getenv("NEO4J_PASS")

	auth := neo4j.BasicAuth(user, pass, "")
	driver, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		logger.Panic(err)
		return nil, err
	}

	// Brza provera konekcije (kao safety)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		logger.Panic(err)
		return nil, err
	}

	return &FollowerRepository{
		driver: driver,
		logger: logger,
	}, nil
}

func (r *FollowerRepository) Close(ctx context.Context) error {
	return r.driver.Close(ctx)
}

// Health: koristi se u Ping da proveri bazu
func (r *FollowerRepository) Health(ctx context.Context) error {
	return r.driver.VerifyConnectivity(ctx)
}

func (r *FollowerRepository) Follow(ctx context.Context, followerID, followeeID string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		const cypher = `
			MATCH (f:User {id: $followerID})
			MATCH (u:User {id: $followeeID})
			MERGE (f)-[r:FOLLOWS]->(u)
			ON CREATE SET r.since = datetime($now)
			RETURN 1 AS ok
		`
		params := map[string]any{
			"followerID": followerID,
			"followeeID": followeeID,
			"now":        time.Now().UTC().Format(time.RFC3339),
		}

		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		// Ako jedan od MATCH-ova ne uspe, neće biti reda u rezultatu
		if !res.Next(ctx) {
			if res.Err() != nil {
				return nil, res.Err()
			}
			return nil, ErrUserNotFound
		}
		return nil, nil
	})
	return err
}

/* — Slede metode koje ćemo dodati kasnije —
func (r *FollowerRepository) Unfollow(ctx context.Context, followerID, followeeID string) error { ... }
func (r *FollowerRepository) ListFollowers(ctx context.Context, userID string, limit, offset int) ([]string, error) { ... }
func (r *FollowerRepository) ListFollowing(ctx context.Context, userID string, limit, offset int) ([]string, error) { ... }
*/

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
