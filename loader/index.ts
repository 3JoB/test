import * as path from 'path';
import * as fs from 'fs';

class LangLoader {
  private langs: Map<string, Map<string, any>> = new Map();

  constructor() {
    this.load('en');
  }

  load(iso = 'en') {
    try {
      const langContent = fs.readFileSync(path.join(__dirname, '../../../lang', `${iso}.yml`), 'utf8');
      const obj = new Map<string, any>();
      this.langs.set(iso, obj);
      let currentObj: Map<string, any> = obj;
      let indent = 0;
      langContent.split('\n').filter(line => line && !line.startsWith('#')).forEach(line => {
        const level = line.match(/^\s+/)![0].length / 2;
        if (level > indent) {
          const key = line.trim().replace(/^\s+/, '').split(':')[0];
          if (line.startsWith('-')) {
            currentObj.set(key, []);
            currentObj = currentObj.get(key);
          } else {
            currentObj.set(key, new Map());
            currentObj = currentObj.get(key);
          }
        } else if (level < indent) {
          for (let i = 0; i < indent - level; i++) currentObj = currentObj.get('parent');
        } else {
          const [k, v] = line.replace(/^\s+/, '').split(':');
          k.trim();
          v.trim();
          currentObj.set(k, v);
        }
        indent = level;
      });
    } catch (e) {
      console.error(e);
    }
  }

  readFile(filePath: string, encoding: string) {
    return fs.readFileSync(filePath, encoding);
  }

  get(iso: string, key: string): any | string {
    const keys = key.split('.');
    const value = this.langs.get(iso);
    if (value == null) return;
    return this.getNestedValue(value, keys, 0);
  }

  private getNestedValue(obj: Map<string, any> | string, keys: string[], index: number): any | string {
    if (!(obj instanceof Map)) return;
    const key = keys[index];
    if (obj.has(key)) {
      if (index === keys.length - 1) {
        return obj.get(key);
      } else {
        return this.getNestedValue(obj.get(key), keys, index + 1);
      }
    }
  }
}

export default LangLoader;
