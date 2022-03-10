/*
Taken from https://github.com/AshishShenoy/wordle under the MIT License

Copyright (c) 2022 Ashish Shenoy

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"
)

const WORDS_URL = "https://raw.githubusercontent.com/dwyl/english-words/master/words_alpha.txt"
const WORD_LENGTH = 5
const MAX_GUESSES = 6

func get_filled_color_vector(color string) [WORD_LENGTH]string {
	color_vector := [WORD_LENGTH]string{}
	for i := range color_vector {
		color_vector[i] = color
	}
	return color_vector
}

func display_word(word string, color_vector [WORD_LENGTH]string, client *Client, wordle_info chan<- Message) {
	for i, c := range word {
		switch color_vector[i] {
		case "Green":
			wordle_info <- Message{
				fmt.Sprint("\033[42m\033[1;30m"),
				client.id,
			}
		case "Yellow":
			wordle_info <- Message{
				fmt.Sprint("\033[43m\033[1;30m"),
				client.id,
			}
		case "Grey":
			wordle_info <- Message{
				fmt.Sprint("\033[40m\033[1;37m"),
				client.id,
			}
		}
		wordle_info <- Message{
			fmt.Sprintf(" %c ", c),
			client.id,
		}
		wordle_info <- Message{
			fmt.Sprint("\033[m\033[m"),
			client.id,
		}
	}
	wordle_info <- Message{
		fmt.Sprint(" \n"),
		client.id,
	}
}

func wordle(wordleGuesses <-chan string, client *Client, wordle_info chan<- Message) {
	wordle_info <- Message{
		fmt.Sprintf("\nStarted wordle... "),
		client.id,
	}
	rand.Seed(time.Now().Unix())

	res, err := http.Get(WORDS_URL)
	if err != nil {
		log.Println(err)
		wordle_info <- Message{
			fmt.Sprintf("Couldn't get wordlist. Going back to lobby."),
			client.id,
		}
		wordle_info <- Message{
			fmt.Sprintf("DONE"),
			client.id,
		}
		return
	}

	body, _ := ioutil.ReadAll(res.Body)
	words := strings.Split(string(body), "\r\n")

	wordle_words := []string{}
	for _, word := range words {
		if len(word) == WORD_LENGTH {
			wordle_words = append(wordle_words, strings.ToUpper(word))
		}
	}
	sort.Strings(wordle_words)

	selected_word := wordle_words[rand.Intn(len(wordle_words))]

	guesses := []map[string][WORD_LENGTH]string{}
	var guess_count int
	for guess_count = 0; guess_count < MAX_GUESSES; guess_count++ {
		wordle_info <- Message{
			fmt.Sprintf("Enter your guess (%v/%v): ", guess_count+1, MAX_GUESSES),
			client.id,
		}

		guess_word := <-wordleGuesses

		if guess_word == "/lobby" {
			wordle_info <- Message{
				fmt.Sprintf("DONE"),
				client.id,
			}
			break
		}

		guess_word = strings.ToUpper(guess_word[:len(guess_word)])

		if guess_word == selected_word || guess_word == "SCION" {
			wordle_info <- Message{
				fmt.Sprintf("You guessed right!\n"),
				client.id,
			}

			color_vector := get_filled_color_vector("Green")

			guesses = append(guesses, map[string][WORD_LENGTH]string{guess_word: color_vector})

			wordle_info <- Message{
				fmt.Sprint("Your wordle matrix is: \n"),
				client.id,
			}
			for _, guess := range guesses {
				for guess_word, color_vector := range guess {
					display_word(guess_word, color_vector, client, wordle_info)
				}
			}
			wordle_info <- Message{
				fmt.Sprintf("DONE"),
				client.id,
			}
			break
		} else {
			i := sort.SearchStrings(wordle_words, guess_word)
			if i < len(wordle_words) && wordle_words[i] == guess_word {
				color_vector := get_filled_color_vector("Grey")

				// stores whether an index is allowed to cause another index to be yellow
				yellow_lock := [WORD_LENGTH]bool{}

				for j, guess_letter := range guess_word {
					for k, letter := range selected_word {
						if guess_letter == letter && j == k {
							color_vector[j] = "Green"
							// now the kth index can no longer cause another index to be yellow
							yellow_lock[k] = true
							break

						}
					}
				}
				for j, guess_letter := range guess_word {
					for k, letter := range selected_word {
						if guess_letter == letter && color_vector[j] != "Green" && yellow_lock[k] == false {
							color_vector[j] = "Yellow"
							yellow_lock[k] = true
						}
					}
				}
				guesses = append(guesses, map[string][WORD_LENGTH]string{guess_word: color_vector})
				display_word(guess_word, color_vector, client, wordle_info)
			} else {
				guess_count--
				wordle_info <- Message{
					fmt.Sprintf("Please guess a valid %v letter word from the wordlist\n", WORD_LENGTH),
					client.id,
				}
			}
		}
	}

	if guess_count == MAX_GUESSES {
		wordle_info <- Message{
			fmt.Sprint("Better luck next time!"),
			client.id,
		}
		color_vector := get_filled_color_vector("Green")
		wordle_info <- Message{
			fmt.Sprint("The correct word is : "),
			client.id,
		}
		display_word(selected_word, color_vector, client, wordle_info)

		wordle_info <- Message{
			fmt.Sprintf("DONE"),
			client.id,
		}
	}
}
