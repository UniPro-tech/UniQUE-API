package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const databaseURL = "api:P@ssw0rd@tcp(localhost:3306)/devdb?charset=utf8&parseTime=True&loc=Local"

func main() {
	fmt.Println("Generating code...")

	dialector := mysql.Open(databaseURL)
	gormdb, err := gorm.Open(dialector, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: false,
		},
	})
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}
	fmt.Println("Connected to database:", databaseURL)

	g := gen.NewGenerator(gen.Config{
		ModelPkgPath: "api/internal/driver/mysql/scheme",
		Mode:         0,
		//Mode:    gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
	})
	g.UseDB(gormdb)

	// Generate models
	g.ApplyBasic(g.GenerateModel("users"))
	g.ApplyBasic(g.GenerateAllTable()...)

	// Write generated code to files
	g.Execute()
}
