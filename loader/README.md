# LangLoader
langloader

## Install

```sh
$ npm i @3job/loader
```

## Use
`lang/en.lang`
```lang
main => {
    test => is test
}
```

index.ts
```ts
import LangLoader from '@3job/loader';

const loader = new LangLoader;
loader.load("en");

console.log(loader.get('main.test')); // output: is test
```