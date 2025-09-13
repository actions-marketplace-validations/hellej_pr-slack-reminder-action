package utilities

import "iter"

func Filter[T any](items []T, filter func(e T) bool) iter.Seq[T] {
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
