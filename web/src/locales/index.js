import { createI18n } from 'vue-i18n' // import from runtime only

import { getLanguage } from '@/utils/cookies'


// User defined lang
import enLocale from './en'
import zhLocale from './zh-cn'

const messages = {
  en: {
    ...enLocale,
  },
  'zh-cn': {
    ...zhLocale,
  }
}

export const getLocale = () => {
  const cookieLanguage = getLanguage()
  if (cookieLanguage) {
    return cookieLanguage
  }
  const language = navigator.language.toLowerCase()
  const locales = Object.keys(messages)
  for (const locale of locales) {
    if (language.indexOf(locale) > -1) {
      return locale
    }
  }

  // Default language is chinese
  return 'zh'
}

const i18n = createI18n({
  locale: getLocale(),
  fallbackLocale: 'zh-cn',
  messages: messages
})

export default i18n