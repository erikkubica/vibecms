import { createContext, useContext, useState, useEffect, type ReactNode } from "react";
import { getLanguages, type Language } from "@/api/client";

interface AdminLanguageContextType {
  languages: Language[];
  currentCode: string;
  currentLanguage: Language | undefined;
  setCurrentCode: (code: string) => void;
}

const AdminLanguageContext = createContext<AdminLanguageContextType>({
  languages: [],
  currentCode: "",
  currentLanguage: undefined,
  setCurrentCode: () => {},
});

const STORAGE_KEY = "squilla_admin_lang";

export function AdminLanguageProvider({ children }: { children: ReactNode }) {
  const [languages, setLanguages] = useState<Language[]>([]);
  const [currentCode, setCurrentCodeState] = useState<string>(() => {
    // Legacy "all" sentinel collapses to "" so the backend resolves it to
    // the site's default language.
    const stored = localStorage.getItem(STORAGE_KEY) || "";
    return stored === "all" ? "" : stored;
  });

  useEffect(() => {
    getLanguages(true).then((langs) => {
      setLanguages(langs);
      const stored = localStorage.getItem(STORAGE_KEY);
      const isInvalid =
        !stored ||
        stored === "all" ||
        !langs.some((l) => l.code === stored);
      if (isInvalid) {
        const fallback = langs.find((l) => l.is_default)?.code || langs[0]?.code || "";
        setCurrentCodeState(fallback);
        if (fallback) localStorage.setItem(STORAGE_KEY, fallback);
        else localStorage.removeItem(STORAGE_KEY);
      }
    }).catch(() => {});
  }, []);

  function setCurrentCode(code: string) {
    setCurrentCodeState(code);
    localStorage.setItem(STORAGE_KEY, code);
  }

  const currentLanguage = languages.find((l) => l.code === currentCode);

  return (
    <AdminLanguageContext.Provider value={{ languages, currentCode, currentLanguage, setCurrentCode }}>
      {children}
    </AdminLanguageContext.Provider>
  );
}

export function useAdminLanguage() {
  return useContext(AdminLanguageContext);
}
