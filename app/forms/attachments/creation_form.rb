# typed: true
# frozen_string_literal: true

require "marcel"

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

    validates :blob_record, presence: true
    validate :validate_file_format
    validate :validate_file_extension
    validate :validate_content_type
    validate :validate_file_content

    # Blobオブジェクトを取得
    sig { returns(T.nilable(ActiveStorage::Blob)) }
    def blob_record
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
      return if blob_record.nil?

      filename = blob_record.not_nil!.filename.to_s

      # 危険なパターンのチェック
      DANGEROUS_FILENAME_PATTERNS.each do |pattern|
        if filename.match?(pattern)
          errors.add(:base, :invalid_filename)
          break
        end
      end
    end

    # 拡張子の検証
    sig { void }
    private def validate_file_extension
      return if blob_record.nil?

      extension = File.extname(blob_record.not_nil!.filename.to_s).downcase
      if EXECUTABLE_EXTENSIONS.include?(extension)
        errors.add(:base, :executable_file_not_allowed)
      end
    end

    # コンテンツタイプの検証
    sig { void }
    private def validate_content_type
      return if blob_record.nil?

      unless Attachments::PresignForm::ALLOWED_CONTENT_TYPES.include?(blob_record.not_nil!.content_type)
        errors.add(:base, :unsupported_file_format)
      end
    end

    # ファイル内容の検証（マジックナンバーチェック）
    sig { void }
    private def validate_file_content
      return if blob_record.nil?

      blob = blob_record.not_nil!

      # ファイルのダウンロードを試みる
      begin
        file_content = blob.download
        return if file_content.blank?

        # Marcelを使用して実際のMIMEタイプを検出
        detected_mime_type = Marcel::MimeType.for(
          file_content,
          name: blob.filename.to_s
        )

        # 検出されたMIMEタイプが許可リストに含まれているか確認
        unless Attachments::PresignForm::ALLOWED_CONTENT_TYPES.include?(detected_mime_type)
          errors.add(:base, :content_type_mismatch)
        end

        # 宣言されたMIMEタイプと実際のMIMEタイプが一致するか確認
        # ただし、一部の互換性のあるMIMEタイプは許可する
        unless compatible_mime_types?(blob.content_type, detected_mime_type)
          errors.add(:base, :content_type_mismatch)
        end
      rescue
        # ダウンロードエラーが発生した場合
        errors.add(:base, :file_download_failed)
      end
    end

    # MIMEタイプの互換性をチェック
    sig { params(declared_type: T.nilable(String), detected_type: String).returns(T::Boolean) }
    private def compatible_mime_types?(declared_type, detected_type)
      return false if declared_type.nil?

      # 完全一致
      return true if declared_type == detected_type

      # 一般的な互換性のあるMIMEタイプのマッピング
      compatible_types = {
        "image/jpg" => "image/jpeg",
        "image/jpeg" => "image/jpg",
        "text/plain" => ["text/x-log", "text/markdown"],
        "application/x-zip-compressed" => "application/zip"
      }

      # 宣言されたタイプと検出されたタイプの互換性をチェック
      if compatible_types[declared_type].is_a?(Array)
        compatible_types[declared_type].include?(detected_type)
      elsif compatible_types[declared_type]
        compatible_types[declared_type] == detected_type
      elsif compatible_types[detected_type].is_a?(Array)
        compatible_types[detected_type].include?(declared_type)
      elsif compatible_types[detected_type]
        compatible_types[detected_type] == declared_type
      else
        false
      end
    end
  end
end
