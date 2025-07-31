# typed: strict
# frozen_string_literal: true

module Attachments
  class CreationForm < ApplicationForm
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

    # Blob署名付きID
    attribute :blob_signed_id, :string

    validates :blob, presence: true
    validate :validate_file_format
    validate :validate_file_extension
    validate :validate_content_type

    # Blobオブジェクトを取得
    sig { returns(T.nilable(ActiveStorage::Blob)) }
    def blob
      return nil if blob_signed_id.blank?

      @blob ||= T.let(
        ActiveStorage::Blob.find_signed(blob_signed_id),
        T.nilable(ActiveStorage::Blob)
      )
    rescue ActiveRecord::RecordNotFound, ActiveSupport::MessageVerifier::InvalidSignature
      nil
    end

    # ファイル名の形式検証
    sig { void }
    private def validate_file_format
      return if blob.nil?

      filename = blob.not_nil!.filename.to_s

      # 危険なパターンのチェック
      DANGEROUS_FILENAME_PATTERNS.each do |pattern|
        if filename.match?(pattern)
          errors.add(:base, "ファイル名が不正です")
          break
        end
      end
    end

    # 拡張子の検証
    sig { void }
    private def validate_file_extension
      return if blob.nil?

      extension = File.extname(blob.not_nil!.filename.to_s).downcase
      if EXECUTABLE_EXTENSIONS.include?(extension)
        errors.add(:base, "実行可能ファイルはアップロードできません")
      end
    end

    # コンテンツタイプの検証
    sig { void }
    private def validate_content_type
      return if blob.nil?

      unless AttachmentPresignForm::ALLOWED_CONTENT_TYPES.include?(blob.not_nil!.content_type)
        errors.add(:base, "サポートされていないファイル形式です")
      end
    end
  end
end
