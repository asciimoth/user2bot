// This file is part of user2bot.
//
// Copyright (c) 2022 AsciiMoth <silkmoth@protonmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// <There will be module docs here>
package user2bot

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/xelaj/mtproto"
	"github.com/xelaj/mtproto/telegram"
)

// TODO Add deafults
type Config struct {
	// Where to store session configuration. Must be set for both bot and userbot
	SessionFile string
	PublicKeysFile string
	// Host address of mtproto server. Actually, it can be any mtproxy, not only official
	// Default: "149.154.167.50:443"
	MTProtoServerHost string
	AppID int64
	AppHash string
	// Default: "Unknown"
	//DeviceModel string
	// Default: "linux/amd64"
	//SystemVersion string
	// Default: "0.1.0"
	//AppVersion string
	PhoneNumber string
}

func SessionFromConfig(config Config) (Session, error) {
	if config.PhoneNumber != "" {
		bot := newUserbot(config)
		return &bot, nil
	}
	// TODO add custom error type with more useful info
	return nil, fmt.Errorf("Session type cannot be determined")
}

type Session interface {
	InitAuth() error
	IsNeedToSendAuthCode() bool
	SendAuthCode(code string) error
	IsNeedToSendPassword() bool
	SendPassword(pass string) error
	InitSession() error
	// RequestNextEvent()
	Close() error
}

type userbot struct {
	config Config
	client *telegram.Client
	setCode *telegram.AuthSentCode
	auth *telegram.AuthAuthorization
	codeAuthNeeded bool
	passAuthNeeded bool
}

func newUserbot(config Config) userbot {
	return userbot{config, nil, nil, nil, false, false}
}

func (u * userbot) InitAuth() error {
	client, err := telegram.NewClient(telegram.ClientConfig{
		SessionFile: u.config.SessionFile,
		ServerHost: u.config.MTProtoServerHost,
		PublicKeysFile:  u.config.PublicKeysFile,
		AppID:           int(u.config.AppID),
		AppHash:         u.config.AppHash,
		InitWarnChannel: false,
	})
	u.client = client
	if err != nil {
		return err
	}
	// Check if we already signed in
	signedIn, err := client.IsSessionRegistred()
	if err != nil {
		return err
	}
	if signedIn {
		return nil
	}
	// Init code auth
	setCode, err := client.AuthSendCode(
		u.config.PhoneNumber,
		int32(u.config.AppID),
		u.config.AppHash, 
		&telegram.CodeSettings{},
	)
	if err != nil {
		return err
	}
	u.setCode = setCode
	u.codeAuthNeeded = true
	return err
}

func (u * userbot) IsNeedToSendAuthCode() bool {
	return u.codeAuthNeeded
}

func (u * userbot) SendAuthCode(code string) error {
	u.codeAuthNeeded = false
	auth, err := u.client.AuthSignIn(
		u.config.PhoneNumber,
		u.setCode.PhoneCodeHash,
		code,
	)
	u.auth = &auth
	if err == nil {
		return err
	}
	errResponse := &mtproto.ErrResponseCode{}
	ok := errors.As(err, &errResponse)
	if !ok || errResponse.Message != "SESSION_PASSWORD_NEEDED" {
		return err
	}
	u.passAuthNeeded = true	
	// TODO autocall SendPassword if password defined in config
	return nil
}

func (u * userbot) IsNeedToSendPassword() bool {
	// We must send password (if needed) only after auth code
	return !u.codeAuthNeeded && u.passAuthNeeded
}

func (u * userbot) SendPassword(pass string) error {
	u.passAuthNeeded = false;
	accountPassword, err := u.client.AccountGetPassword()
	if err != nil {
		return err
	}
	inputCheck, err := telegram.GetInputCheckPassword(pass, accountPassword)
	if err != nil {
		return err
	}
	auth, err := u.client.AuthCheckPassword(inputCheck)
	u.auth = &auth
	return err
}

func (u * userbot) InitSession() error {
	// See https://github.com/xelaj/mtproto#what-is-invokewithlayer for explanations
	/*_, err := u.client.InvokeWithLayer(apiVersion, &telegram.InitConnectionParams{
		ApiID:          124100,
		DeviceModel:    u.config.DeviceModel,
		SystemVersion:  u.config.SystemVersion,
		AppVersion:     u.config.AppVersion,
		// just use "en", any other language codes will receive error. See telegram docs for more info.
		SystemLangCode: "en",
		LangCode:       "en",
		// HelpGetConfig() is ACTUAL request, but wrapped in InvokeWithLayer
		Query:          &telegram.HelpGetConfigParams{},
	})
	return err
	*/
	return nil
}

func (u * userbot) Close() error {
	//TODO
	return nil
}
