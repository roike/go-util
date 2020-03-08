package helper

import (
	"math/rand"
	"time"
)

var randSrc = rand.NewSource(time.Now().UnixNano())

const (
	alphabets = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	idxBits   = 6
	idxMask   = 1<<idxBits - 1 //bitをすべて1にする
	idxMax    = 63 / 6
)

/*
乱数を使って指定文字数のランダムな文字列を生成する
生成する文字列はアルファベットの52文字を使う
従って0-51の範囲で乱数が生成されれば良い-->range len(alphabets)
一般にrandSrc.Int63で生成される乱数のうち下位の6ビットが
len(alphabets)を超える確率は小さいため5ビットでマスク(int(cache & 6)できる
このマスクによってrandSrc.Int63で一度生成した乱数は(cache >>=6)bit演算回数使いまわせる
(randSrc.Int63)を有効活用して全体の計算回数(randSrc.Int63)を減らす
*/
func RandomString(n int) string {
	b := make([]byte, n)
	l := len(alphabets)
	// A randSrc.Int63() generates 63 random bits, enough for idxMax letters!
	for i, cache, remain := n-1, randSrc.Int63(), idxMax; i >= 0; {
		if remain == 0 {
			cache, remain = randSrc.Int63(), idxMax
		}
		if idx := int(cache & idxMask); idx < l {
			b[i] = alphabets[idx]
			i--
		}
		cache >>= idxBits
		remain--
	}

	return string(b)
}
