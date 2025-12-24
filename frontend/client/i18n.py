import json
import os

class I18n:
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super(I18n, cls).__new__(cls)
            cls._instance.lang = "zh"
            cls._instance.data = {}
            cls._instance.load_locales()
        return cls._instance

    def load_locales(self):
        # Assumes running from 'frontend/' directory or project root
        base_path = os.path.join("assets", "locales")
        # Try finding assets folder
        if not os.path.exists(base_path):
             # Try going up one level if we are in frontend/client/
             base_path = os.path.join("..", "assets", "locales")
        
        # Fallback to absolute relative to this file
        if not os.path.exists(base_path):
             base_path = os.path.join(os.path.dirname(__file__), "..", "..", "assets", "locales")

        for lang in ["en", "zh"]:
            path = os.path.join(base_path, f"{lang}.json")
            try:
                with open(path, "r", encoding="utf-8") as f:
                    self.data[lang] = json.load(f)
            except Exception as e:
                print(f"Error loading locale {lang}: {e}")
                self.data[lang] = {}

    def set_lang(self, lang):
        if lang in self.data:
            self.lang = lang

    def t(self, key):
        return self.data.get(self.lang, {}).get(key, key)

    def get_list(self, key):
        # For manual lines etc.
        val = self.data.get(self.lang, {}).get(key, [])
        if isinstance(val, list):
            return val
        return []

# Singleton instance
i18n = I18n()
