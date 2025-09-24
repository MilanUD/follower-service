package repo

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"database-example/model"

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

var (
	ErrNotFollowing = errors.New("follow relationship does not exist")
)

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

func (r *FollowerRepository) Unfollow(ctx context.Context, followerID, followeeID string) error {
	// zaštita od self-unfollow; može i u servisu ako hoćeš
	if followerID == followeeID {
		return errors.New("cannot unfollow self")
	}

	ses := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer ses.Close(ctx)

	_, err := ses.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// ako nema takve relacije, deleted=0
		res, err := tx.Run(ctx, `
			MATCH (:User {id:$followerID})-[r:FOLLOWS]->(:User {id:$followeeID})
			DELETE r
			RETURN COUNT(r) AS deleted
		`, map[string]any{
			"followerID": followerID,
			"followeeID": followeeID,
		})
		if err != nil {
			return nil, err
		}
		rec, err := res.Single(ctx)
		if err != nil {
			return nil, err
		}
		deleted, _ := rec.Get("deleted")
		if n, ok := deleted.(int64); !ok || n == 0 {
			return nil, ErrNotFollowing
		}
		return nil, nil
	})
	return err
}

func (r *FollowerRepository) GetRecommendations(ctx context.Context, userID string, limit int) ([]model.Recommendation, error) {
	if limit <= 0 {
		limit = 10
	}

	ses := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer ses.Close(ctx)

	recsAny, err := ses.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
            MATCH (me:User {id: $userId})-[:FOLLOWS]->(:User)-[:FOLLOWS]->(cand:User)
            WHERE cand.id <> $userId
              AND NOT (me)-[:FOLLOWS]->(cand)
            WITH cand, count(*) AS mutual
            RETURN cand.id AS user_id, mutual
            ORDER BY mutual DESC
            LIMIT $limit
        `, map[string]any{"userId": userID, "limit": limit})
		if err != nil {
			return nil, err
		}

		out := make([]model.Recommendation, 0)
		for res.Next(ctx) {
			rec := res.Record()
			id, _ := rec.Get("user_id")
			mutual, _ := rec.Get("mutual")
			out = append(out, model.Recommendation{
				UserID: id.(string),
				Mutual: mutual.(int64),
			})
		}
		return out, res.Err()
	})
	if err != nil {
		return nil, err
	}
	return recsAny.([]model.Recommendation), nil
}

func (r *FollowerRepository) GetFollowees(ctx context.Context, userID string, skip, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	if skip < 0 {
		skip = 0
	}

	ses := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer ses.Close(ctx)

	resAny, err := ses.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (:User {id:$userId})-[:FOLLOWS]->(f:User)
			RETURN f.id AS id
			ORDER BY id
			SKIP $skip LIMIT $limit
		`, map[string]any{
			"userId": userID,
			"skip":   skip,
			"limit":  limit,
		})
		if err != nil {
			return nil, err
		}

		out := make([]string, 0)
		for res.Next(ctx) {
			idVal, _ := res.Record().Get("id")
			out = append(out, idVal.(string))
		}
		return out, res.Err()
	})
	if err != nil {
		return nil, err
	}
	return resAny.([]string), nil
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
