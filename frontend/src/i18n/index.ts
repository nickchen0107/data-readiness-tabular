import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import HttpBackend from 'i18next-http-backend'
import zhTW from './fallback/zh-TW.json'
import en from './fallback/en.json'

const savedLang = localStorage.getItem('language') || 'zh-TW'

i18n
  .use(HttpBackend)
  .use(initReactI18next)
  .init({
    lng: savedLang,
    fallbackLng: 'zh-TW',
    ns: ['translation'],
    defaultNS: 'translation',
    backend: {
      loadPath: '/api/translations/{{lng}}',
      parse: (data: string) => {
        const parsed = JSON.parse(data)
        return parsed.translations || parsed
      },
    },
    partialBundledLanguages: true,
    resources: {
      'zh-TW': { translation: zhTW },
      en: { translation: en },
    },
    interpolation: {
      escapeValue: false,
    },
    react: {
      useSuspense: false,
    },
  })

export default i18n
