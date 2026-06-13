//go:build ignore

// A deliberately rough snippet to demo the reviewer (excluded from the build).
package main

import "fmt"

func Process(data []map[string]interface{}) []string {
	result := []string{}
	for i := 0; i < len(data); i++ {
		if data[i] != nil {
			if data[i]["status"] == "active" {
				name := data[i]["name"].(string)
				result = append(result, name)
			}
		}
	}
	return result
}

func main() {
	fmt.Println(Process(nil))
}
