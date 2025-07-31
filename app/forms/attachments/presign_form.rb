# typed: strict
# frozen_string_literal: true

module Attachments
  class PresignForm < ApplicationForm
    # ファイルタイプ別の最大サイズ
    IMAGE_MAX_SIZE = T.let(10.megabytes, Integer)
    VIDEO_MAX_SIZE = T.let(100.megabytes, Integer)
    OTHER_MAX_SIZE = T.let(25.megabytes, Integer)

    # ファイル名の最大バイト数
    MAX_FILENAME_BYTES = T.let(255, Integer)

    # 画像のMIMEタイプ
    IMAGE_CONTENT_TYPES = T.let([
      "image/jpeg",
      "image/jpg",
      "image/png",
      "image/gif",
      "image/svg+xml",
      "image/webp"
    ].freeze, T::Array[String])

    # 動画のMIMEタイプ
    VIDEO_CONTENT_TYPES = T.let([
      "video/mp4",
      "video/webm",
      "video/quicktime" # .mov
    ].freeze, T::Array[String])

    # その他のMIMEタイプ
    OTHER_CONTENT_TYPES = T.let([
      # ドキュメント
      "application/pdf",
      "application/vnd.openxmlformats-officedocument.wordprocessingml.document", # .docx
      "application/vnd.openxmlformats-officedocument.presentationml.presentation", # .pptx
      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", # .xlsx
      "application/vnd.ms-excel", # .xls
      # テキストファイル
      "text/plain",
      "text/csv",
      "text/x-log",
      "text/markdown",
      "application/json",
      # アーカイブ
      "application/zip",
      "application/x-tar",
      "application/gzip",
      "application/x-compressed-tar" # .tgz
    ].freeze, T::Array[String])

    # 許可する全てのMIMEタイプ
    ALLOWED_CONTENT_TYPES = T.let(
      (IMAGE_CONTENT_TYPES + VIDEO_CONTENT_TYPES + OTHER_CONTENT_TYPES).freeze,
      T::Array[String]
    )

    attribute :filename, :string
    attribute :content_type, :string
    attribute :byte_size, :integer

    validates :filename, presence: true
    validates :content_type, presence: true, inclusion: {
      in: ALLOWED_CONTENT_TYPES,
      message: :unsupported_content_type
    }
    validates :byte_size, presence: true

    validate :validate_filename_format
    validate :validate_file_size

    # ファイル名のバリデーション
    sig { void }
    private def validate_filename_format
      return if filename.blank?

      fname = filename.not_nil!

      # Unicode正規化（NFC）
      normalized_fname = fname.unicode_normalize(:nfc)
      if fname != normalized_fname
        self.filename = normalized_fname
        fname = normalized_fname
      end

      # 危険な文字が含まれていないかチェック
      if fname.match?(/[<>:"|?*\x00-\x1f]/)
        errors.add(:filename, :invalid_characters)
      end

      # パストラバーサルの防止
      if fname.include?("..") || fname.include?("/") || fname.include?("\\")
        errors.add(:filename, :invalid_path)
      end

      # ファイル名の長さ制限（255バイト）
      if fname.bytesize > MAX_FILENAME_BYTES
        errors.add(:filename, :too_long)
      end
    end

    # ファイルサイズのバリデーション
    sig { void }
    private def validate_file_size
      return if byte_size.blank? || content_type.blank?

      size = byte_size.not_nil!
      type = content_type.not_nil!

      # ファイルサイズが0以下の場合
      if size <= 0
        errors.add(:byte_size, :greater_than, count: 0)
        return
      end

      # ファイルタイプ別のサイズ制限
      max_size = case type
      when *IMAGE_CONTENT_TYPES
        IMAGE_MAX_SIZE
      when *VIDEO_CONTENT_TYPES
        VIDEO_MAX_SIZE
      else
        OTHER_MAX_SIZE
      end

      if size > max_size
        errors.add(:byte_size, :file_size_too_large, max: ActiveSupport::NumberHelper.number_to_human_size(max_size))
      end
    end
  end
end
