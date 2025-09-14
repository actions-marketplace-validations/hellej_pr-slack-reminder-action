package utilities

import (
	"iter"
	"slices"
)

func Filter[T any](items []T, filter func(e T) bool) []T {
	return slices.Collect(FilterToIter(items, filter))
}

func FilterToIter[T any](items []T, filter func(e T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, item := range items {

			if !filter(item) {
				continue
			}

			if !yield(item) {
				return
			}

		}
	}
}

func Map[T any, V any](items []T, mapper func(T) V) []V {
	return slices.Collect(MapToIter(items, mapper))
}

func MapToIter[T any, V any](items []T, mapper func(T) V) iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, item := range items {
			if !yield(mapper(item)) {
				return
			}
		}
	}
}

// exits early on error (and returns it)
func MapWithError[T any, V any](items []T, mapper func(T) (V, error)) ([]V, error) {
	var result []V
	var firstError error

	for mapped, err := range MapWithErrorToIter(items, mapper) {
		if err != nil {
			firstError = err
			break
		}
		result = append(result, mapped)
	}

	return result, firstError
}

// exits early on error (and returns it)
func MapWithErrorToIter[T any, V any](items []T, mapper func(T) (V, error)) iter.Seq2[V, error] {
	return func(yield func(V, error) bool) {
		for _, item := range items {
			mapped, err := mapper(item)
			if !yield(mapped, err) {
				return
			}
			if err != nil {
				return
			}
		}
	}
}
