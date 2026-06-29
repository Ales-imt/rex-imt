package service

import "time"

// AcademicYear retourne l'année de début de l'année académique courante.
// Juillet marque le début de la nouvelle année.
// Ex : janvier-juin 2025 → 2024 ; juillet-décembre 2025 → 2025.
func AcademicYear(now time.Time) int32 {
	if now.Month() >= time.July {
		return int32(now.Year())
	}
	return int32(now.Year() - 1)
}
