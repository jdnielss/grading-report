package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	router := gin.Default()

	// Ensure the database directory exists
	if _, err := os.Stat("./database"); os.IsNotExist(err) {
		os.Mkdir("./database", os.ModePerm)
	}

	// Initialize SQLite database
	db, err := sql.Open("sqlite3", "./database/database.sqlite")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create reports table if it doesn't exist
	createTableQuery := `
    CREATE TABLE IF NOT EXISTS reports (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user TEXT,
        project TEXT,
        new_reliability_rating TEXT,
        new_security_rating TEXT,
        new_maintainability_rating TEXT,
        bugs TEXT,
        code_smells TEXT,
        critical_violations TEXT,
        uncovered_lines TEXT,
        result TEXT
    );`
	if _, err := db.Exec(createTableQuery); err != nil {
		log.Fatal(err)
	}

	// POST /report endpoint
	router.POST("/report", func(c *gin.Context) {
		var report struct {
			User                     string `json:"user"`
			Project                  string `json:"project"`
			NewReliabilityRating     string `json:"new_reliability_rating"`
			NewSecurityRating        string `json:"new_security_rating"`
			NewMaintainabilityRating string `json:"new_maintainability_rating"`
			Bugs                     string `json:"bugs"`
			CodeSmells               string `json:"code_smells"`
			CriticalViolations       string `json:"critical_violations"`
			UncoveredLines           string `json:"uncovered_lines"`
		}

		if err := c.BindJSON(&report); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Calculate total "OK" values
		ratings := []string{
			report.NewReliabilityRating,
			report.NewSecurityRating,
			report.NewMaintainabilityRating,
			report.Bugs,
			report.CodeSmells,
			report.CriticalViolations,
		}

		totalOK := 0
		for _, rating := range ratings {
			if path.Ext(rating) == "OK" {
				totalOK++
			}
		}

		fmt.Println(totalOK)

		// Determine result based on totalOK
		result := "FAIL"
		if totalOK >= 6 {
			result = "PASS"
		}

		// Insert report data into SQLite database
		stmt, err := db.Prepare(`
            INSERT INTO reports (
                user, project, new_reliability_rating, new_security_rating,
                new_maintainability_rating, bugs, code_smells,
                critical_violations, uncovered_lines, result
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_, err = stmt.Exec(
			report.User, report.Project, report.NewReliabilityRating,
			report.NewSecurityRating, report.NewMaintainabilityRating,
			report.Bugs, report.CodeSmells, report.CriticalViolations,
			report.UncoveredLines, result)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.String(http.StatusOK, "Report saved successfully!")
	})

	// GET /report/:user endpoint
	router.GET("/report/:user", func(c *gin.Context) {
		user := c.Param("user")

		rows, err := db.Query(`SELECT project, new_reliability_rating, new_security_rating, new_maintainability_rating, bugs, code_smells, critical_violations, uncovered_lines, result FROM reports WHERE user = ?`, user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var reports []map[string]string
		for rows.Next() {
			var project, newReliabilityRating, newSecurityRating, newMaintainabilityRating, bugs, codeSmells, criticalViolations, uncoveredLines, result string
			if err := rows.Scan(&project, &newReliabilityRating, &newSecurityRating, &newMaintainabilityRating, &bugs, &codeSmells, &criticalViolations, &uncoveredLines, &result); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			reports = append(reports, map[string]string{
				"project":                    project,
				"new_reliability_rating":     newReliabilityRating,
				"new_security_rating":        newSecurityRating,
				"new_maintainability_rating": newMaintainabilityRating,
				"bugs":                       bugs,
				"code_smells":                codeSmells,
				"critical_violations":        criticalViolations,
				"uncovered_lines":            uncoveredLines,
				"result":                     result,
			})
		}

		if len(reports) == 0 {
			c.String(http.StatusNotFound, "No reports found for user %s", user)
			return
		}

		html := `
        <html>
          <head>
            <title>Report for ` + user + `</title>
            <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css" rel="stylesheet">
          </head>
          <body>
            <div class="container mx-auto">
              <h2 class="text-2xl mb-4">Report for ` + user + `</h2>
              <table class="table-auto w-full">
                <thead>
                  <tr>
                    <th class="px-4 py-2">Project</th>
                    <th class="px-4 py-2">New Reliability Rating</th>
                    <th class="px-4 py-2">New Security Rating</th>
                    <th class="px-4 py-2">New Maintainability Rating</th>
                    <th class="px-4 py-2">Bugs</th>
                    <th class="px-4 py-2">Code Smells</th>
                    <th class="px-4 py-2">Critical Violations</th>
                    <th class="px-4 py-2">Uncovered Lines</th>
                    <th class="px-4 py-2">Result</th>
                  </tr>
                </thead>
                <tbody>
        `
		for _, report := range reports {
			html += `
            <tr>
              <td class="border px-4 py-2">` + report["project"] + `</td>
              <td class="border px-4 py-2">` + report["new_reliability_rating"] + `</td>
              <td class="border px-4 py-2">` + report["new_security_rating"] + `</td>
              <td class="border px-4 py-2">` + report["new_maintainability_rating"] + `</td>
              <td class="border px-4 py-2">` + report["bugs"] + `</td>
              <td class="border px-4 py-2">` + report["code_smells"] + `</td>
              <td class="border px-4 py-2">` + report["critical_violations"] + `</td>
              <td class="border px-4 py-2">` + report["uncovered_lines"] + `</td>
              <td class="border px-4 py-2">` + report["result"] + `</td>
            </tr>
            `
		}
		html += `
                </tbody>
              </table>
            </div>
          </body>
        </html>
        `
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	})

	router.Run(":3000")
}
