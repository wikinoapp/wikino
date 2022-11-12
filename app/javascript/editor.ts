import Trix from 'trix';

document.addEventListener('trix-before-initialize', () => {
  // ツールバーを無効化する
  Trix.config.toolbar.getDefaultHTML = () => null;
});
