# typed: strict
# frozen_string_literal: true

class AttachmentValidationService < ApplicationService
  extend T::Sig

  # 実行可能ファイルの拡張子
  EXECUTABLE_EXTENSIONS = T.let(%w[
    .exe .bat .cmd .com .scr .vbs .vbe .js .jse .wsf .wsh .msi .jar .app
    .deb .rpm .dmg .pkg .run .sh .bash .zsh .fish .ps1 .psm1 .psd1 .ps1xml
    .psc1 .psc2
  ].freeze, T::Array[String])

  # 危険なファイル名パターン
  DANGEROUS_FILENAME_PATTERNS = T.let([
    /\A\./, # 隠しファイル
    /\.{2,}/, # 連続するドット（パストラバーサル）
    /[<>:"|?*\x00-\x1f]/, # 制御文字や危険な文字
    %r{[/\\]} # パス区切り文字
  ].freeze, T::Array[Regexp])

  sig { params(blob: ActiveStorage::Blob).returns(T::Boolean) }
  def self.valid?(blob)
    new(blob: blob).valid?
  end

  sig { params(blob: ActiveStorage::Blob).void }
  def initialize(blob:)
    @blob = blob
  end

  sig { returns(T::Boolean) }
  def valid?
    valid_filename? && valid_extension? && valid_content_type?
  end

  sig { returns(T::Array[String]) }
  def errors
    errors = []
    errors << "ファイル名が不正です" unless valid_filename?
    errors << "実行可能ファイルはアップロードできません" unless valid_extension?
    errors << "サポートされていないファイル形式です" unless valid_content_type?
    errors
  end

  private

  sig { returns(ActiveStorage::Blob) }
  attr_reader :blob

  # ファイル名の検証
  sig { returns(T::Boolean) }
  private def valid_filename?
    filename = blob.filename.to_s

    # 危険なパターンのチェック
    DANGEROUS_FILENAME_PATTERNS.none? { |pattern| filename.match?(pattern) }
  end

  # 拡張子の検証
  sig { returns(T::Boolean) }
  private def valid_extension?
    extension = File.extname(blob.filename.to_s).downcase
    !EXECUTABLE_EXTENSIONS.include?(extension)
  end

  # コンテンツタイプの検証
  sig { returns(T::Boolean) }
  private def valid_content_type?
    # AttachmentPresignFormで定義された許可リストを使用
    AttachmentPresignForm::ALLOWED_CONTENT_TYPES.include?(blob.content_type)
  end
end
