package utils

import (
	"database/sql"
	"fmt"

	"log/slog"

	_ "modernc.org/sqlite" // The underscore registers the driver without directly using it
)

const RpmDBPath = "/var/lib/rpm/rpmdb.sqlite"

func connectToRpmSQLiteDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", RpmDBPath)
	if err != nil {
		slog.Error("failed opening rpm db", "err", err)
		return nil, err
	}
	return db, nil
}

func getRpmSQLiteDBPackageData(db *sql.DB, packageName string) (int, []byte, error) {
	var hnum int
	var blob []byte
	packageQuery := fmt.Sprintf("SELECT Packages.hnum, Packages.blob FROM Packages INNER JOIN Name on Name.hnum = Packages.hnum WHERE Name.key = '%s'", packageName)
	err := db.QueryRow(packageQuery).Scan(&hnum, &blob)
	if err != nil {
		slog.Error("failed querying rpm db", "err", err)
		return 0, nil, err
	}
	return hnum, blob, nil
}

func updatePackageSQLite(db *sql.DB, hnum int, newBlob []byte, newName string) error {
	updateBlobQuery := "UPDATE Packages SET blob = ? WHERE hnum = ?"
	_, err := db.Exec(updateBlobQuery, newBlob, hnum)
	if err != nil {
		slog.Error("failed updating rpm db blob", "err", err)
		return err
	}

	updateNameQuery := "UPDATE Name SET key = ? WHERE hnum = ?"
	_, err = db.Exec(updateNameQuery, newName, hnum)
	if err != nil {
		slog.Error("failed updating rpm db name", "err", err)
		return err
	}

	deleteQuery := "DELETE FROM Sha1header WHERE hnum = ?"
	_, err = db.Exec(deleteQuery, hnum)
	if err != nil {
		slog.Error("failed deleting rpm db sha1", "err", err)
		return err
	}

	return nil
}
