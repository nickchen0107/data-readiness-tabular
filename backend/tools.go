//go:build tools

package tools

// This file imports packages to ensure they remain in go.mod.
// These dependencies are required by the project but not yet used in main.
import (
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/google/uuid"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/stretchr/testify/assert"
	_ "github.com/xuri/excelize/v2"
	_ "golang.org/x/crypto/bcrypt"
	_ "pgregory.net/rapid"
)
