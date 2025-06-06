// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package i18n

import (
	"fmt"
	"html/template"
	"slices"

	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"
)

// This file implements the static LocaleStore that will not watch for changes

type locale struct {
	store       *localeStore
	langName    string
	idxToMsgMap map[int]string // the map idx is generated by store's trKeyToIdxMap

	newStyleMessages map[string]string
	pluralRule       PluralFormRule
}

var _ Locale = (*locale)(nil)

type localeStore struct {
	// After initializing has finished, these fields are read-only.
	langNames []string
	langDescs []string

	localeMap     map[string]*locale
	trKeyToIdxMap map[string]int

	defaultLang string
}

// NewLocaleStore creates a static locale store
func NewLocaleStore() LocaleStore {
	return &localeStore{localeMap: make(map[string]*locale), trKeyToIdxMap: make(map[string]int)}
}

const (
	PluralFormSeparator string = "\036"
)

// A note about pluralization rules.
// go-i18n supports plural rules in theory.
// In practice, it relies on another library that hardcodes a list of common languages
// and their plural rules, and does not support languages not hardcoded there.
// So we pretend that all languages are English and use our own function to extract
// the correct plural form for a given count and language.

// AddLocaleByIni adds locale by ini into the store
func (store *localeStore) AddLocaleByIni(langName, langDesc string, pluralRule PluralFormRule, source, moreSource []byte) error {
	if _, ok := store.localeMap[langName]; ok {
		return ErrLocaleAlreadyExist
	}

	store.langNames = append(store.langNames, langName)
	store.langDescs = append(store.langDescs, langDesc)

	l := &locale{store: store, langName: langName, idxToMsgMap: make(map[int]string), pluralRule: pluralRule, newStyleMessages: make(map[string]string)}
	store.localeMap[l.langName] = l

	iniFile, err := setting.NewConfigProviderForLocale(source, moreSource)
	if err != nil {
		return fmt.Errorf("unable to load ini: %w", err)
	}

	for _, section := range iniFile.Sections() {
		for _, key := range section.Keys() {
			var trKey string
			// see https://codeberg.org/forgejo/discussions/issues/104
			//     https://github.com/WeblateOrg/weblate/issues/10831
			// for an explanation of why "common" is an alternative
			if section.Name() == "" || section.Name() == "DEFAULT" || section.Name() == "common" {
				trKey = key.Name()
			} else {
				trKey = section.Name() + "." + key.Name()
			}
			idx, ok := store.trKeyToIdxMap[trKey]
			if !ok {
				idx = len(store.trKeyToIdxMap)
				store.trKeyToIdxMap[trKey] = idx
			}
			l.idxToMsgMap[idx] = key.Value()
		}
	}

	return nil
}

func RecursivelyAddTranslationsFromJSON(locale *locale, object map[string]any, prefix string) error {
	for key, value := range object {
		var fullkey string
		if prefix != "" {
			fullkey = prefix + "." + key
		} else {
			fullkey = key
		}

		switch v := value.(type) {
		case string:
			// Check whether we are adding a plural form to the parent object, or a new nested JSON object.

			if key == "zero" || key == "one" || key == "two" || key == "few" || key == "many" {
				locale.newStyleMessages[prefix+PluralFormSeparator+key] = v
			} else if key == "other" {
				locale.newStyleMessages[prefix] = v
			} else {
				locale.newStyleMessages[fullkey] = v
			}

		case map[string]any:
			err := RecursivelyAddTranslationsFromJSON(locale, v, fullkey)
			if err != nil {
				return err
			}

		case nil:
		default:
			return fmt.Errorf("Unrecognized JSON value '%s'", value)
		}
	}

	return nil
}

func (store *localeStore) AddToLocaleFromJSON(langName string, source []byte) error {
	locale, ok := store.localeMap[langName]
	if !ok {
		return ErrLocaleDoesNotExist
	}

	var result map[string]any
	if err := json.Unmarshal(source, &result); err != nil {
		return err
	}

	return RecursivelyAddTranslationsFromJSON(locale, result, "")
}

func (l *locale) LookupNewStyleMessage(trKey string) string {
	if msg, ok := l.newStyleMessages[trKey]; ok {
		return msg
	}
	return ""
}

func (l *locale) LookupPlural(trKey string, count any) string {
	n, err := util.ToInt64(count)
	if err != nil {
		log.Error("Invalid plural count '%s'", count)
		return ""
	}

	pluralForm := l.pluralRule(n)
	suffix := ""
	switch pluralForm {
	case PluralFormZero:
		suffix = PluralFormSeparator + "zero"
	case PluralFormOne:
		suffix = PluralFormSeparator + "one"
	case PluralFormTwo:
		suffix = PluralFormSeparator + "two"
	case PluralFormFew:
		suffix = PluralFormSeparator + "few"
	case PluralFormMany:
		suffix = PluralFormSeparator + "many"
	case PluralFormOther:
		// No suffix for the "other" string.
	default:
		log.Error("Invalid plural form index %d for count %d", pluralForm, count)
		return ""
	}

	if result, ok := l.newStyleMessages[trKey+suffix]; ok {
		return result
	}

	log.Error("Missing translation for plural form index %d for count %d", pluralForm, count)
	return ""
}

func (store *localeStore) HasLang(langName string) bool {
	_, ok := store.localeMap[langName]
	return ok
}

func (store *localeStore) ListLangNameDesc() (names, desc []string) {
	return store.langNames, store.langDescs
}

// SetDefaultLang sets default language as a fallback
func (store *localeStore) SetDefaultLang(lang string) {
	store.defaultLang = lang
}

// Locale returns the locale for the lang or the default language
func (store *localeStore) Locale(lang string) (Locale, bool) {
	l, found := store.localeMap[lang]
	if !found {
		var ok bool
		l, ok = store.localeMap[store.defaultLang]
		if !ok {
			// no default - return an empty locale
			l = &locale{store: store, idxToMsgMap: make(map[int]string)}
		}
	}
	return l, found
}

func (store *localeStore) Close() error {
	return nil
}

func (l *locale) TrString(trKey string, trArgs ...any) string {
	format := trKey

	if msg := l.LookupNewStyleMessage(trKey); msg != "" {
		format = msg
	} else {
		// First fallback: old-style translation
		idx, foundIndex := l.store.trKeyToIdxMap[trKey]
		found := false
		if foundIndex {
			if msg, ok := l.idxToMsgMap[idx]; ok {
				format = msg // use the found translation
				found = true
			}
		}

		if !found {
			// Second fallback: new-style default language
			if defaultLang, ok := l.store.localeMap[l.store.defaultLang]; ok {
				if msg := defaultLang.LookupNewStyleMessage(trKey); msg != "" {
					format = msg
					found = true
				} else if foundIndex {
					// Third fallback: old-style default language
					if msg, ok := defaultLang.idxToMsgMap[idx]; ok {
						format = msg
						found = true
					}
				}
			}

			if !found {
				log.Error("Missing translation %q", trKey)
			}
		}
	}

	msg, err := Format(format, trArgs...)
	if err != nil {
		log.Error("Error whilst formatting %q in %s: %v", trKey, l.langName, err)
	}
	return msg
}

func PrepareArgsForHTML(trArgs ...any) []any {
	args := slices.Clone(trArgs)
	for i, v := range args {
		switch v := v.(type) {
		case nil, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, template.HTML:
			// for most basic types (including template.HTML which is safe), just do nothing and use it
		case string:
			args[i] = template.HTMLEscapeString(v)
		case fmt.Stringer:
			args[i] = template.HTMLEscapeString(v.String())
		default:
			args[i] = template.HTMLEscapeString(fmt.Sprint(v))
		}
	}
	return args
}

func (l *locale) TrHTML(trKey string, trArgs ...any) template.HTML {
	return template.HTML(l.TrString(trKey, PrepareArgsForHTML(trArgs...)...))
}

func (l *locale) TrPluralString(count any, trKey string, trArgs ...any) template.HTML {
	message := l.LookupPlural(trKey, count)

	if message == "" {
		if defaultLang, ok := l.store.localeMap[l.store.defaultLang]; ok {
			message = defaultLang.LookupPlural(trKey, count)
		}
		if message == "" {
			message = trKey
		}
	}

	message, err := Format(message, PrepareArgsForHTML(trArgs...)...)
	if err != nil {
		log.Error("Error whilst formatting %q in %s: %v", trKey, l.langName, err)
	}
	return template.HTML(message)
}

// HasKey returns whether a key is present in this locale or not
func (l *locale) HasKey(trKey string) bool {
	_, ok := l.newStyleMessages[trKey]
	if ok {
		return true
	}
	idx, ok := l.store.trKeyToIdxMap[trKey]
	if !ok {
		return false
	}
	_, ok = l.idxToMsgMap[idx]
	return ok
}
