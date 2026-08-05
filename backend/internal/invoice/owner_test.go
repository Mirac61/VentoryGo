package invoice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// Feste Owner fuer die Tests gegen das In-Memory-Repo, das keinen Foreign Key
// kennt. Die Postgres-Tests brauchen echte users-Zeilen und nehmen seedUser.
var (
	testOwnerID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testOwner    = testOwnerID.String()
	otherOwnerID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	otherOwner   = otherOwnerID.String()
)

// seedUser legt einen Nutzer an und gibt seine ID zurueck. invoices.owner_id
// haengt per Foreign Key an users, ein erfundener String reicht dort nicht.
// Das Aufraeumen nimmt die Rechnungen des Nutzers per ON DELETE CASCADE mit.
func seedUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	id := uuid.NewString()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES ($1, $2, 'x', now())
	`, id, id+"@example.test")
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, id)
	})
	return id
}
