# typed: strict
# frozen_string_literal: true

module Attachments
  class PresignForm < ApplicationForm
    # 最大ファイルサイズ: 50MB
    MAX_FILE_SIZE = T.let(50.megabytes, Integer)

    # 許可するMIMEタイプ
    ALLOWED_CONTENT_TYPES = T.let([
      # 画像
      "image/jpeg",
      "image/jpg",
      "image/png",
      "image/gif",
      "image/svg+xml",
      "image/webp",
      # ドキュメント
      "application/pdf",
      "text/plain",
      "text/csv",
      # アーカイブ
      "application/zip",
      "application/x-tar",
      "application/gzip"
    ].freeze, T::Array[String])

    attribute :filename, :string
    attribute :content_type, :string
    attribute :byte_size, :integer

    validates :filename, presence: true
    validates :content_type, presence: true, inclusion: {
      in: ALLOWED_CONTENT_TYPES,
      message: :unsupported_content_type
    }
    validates :byte_size, presence: true, numericality: {
      greater_than: 0,
      less_than_or_equal_to: MAX_FILE_SIZE,
      message: :file_size_too_large
    }

    validate :validate_filename_format

    # ファイル名のサニタイズ
    sig { void }
    private def validate_filename_format
      return if filename.blank?

      fname = filename.not_nil!

      # 危険な文字が含まれていないかチェック
      if fname.match?(/[<>:"|?*\x00-\x1f]/)
        errors.add(:filename, :invalid_characters)
      end

      # パストラバーサルの防止
      if fname.include?("..") || fname.include?("/") || fname.include?("\\")
        errors.add(:filename, :invalid_path)
      end
    end
  end
end
