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
    attribute :checksum, :string

    # コンストラクタをオーバーライド
    sig { params(attributes: T.untyped).void }
    def initialize(attributes = {})
      super
      sanitize_filename
    end

    validates :filename, presence: true
    validates :content_type, presence: true, inclusion: {
      in: ALLOWED_CONTENT_TYPES,
      message: :unsupported_content_type
    }
    validates :byte_size, presence: true
    validates :checksum, presence: true

    validate :validate_filename_format
    validate :validate_file_size

    # ファイル名のサニタイズ処理
    sig { void }
    private def sanitize_filename
      return if filename.blank?

      fname = filename.not_nil!

      # Unicode正規化（NFC）
      fname = fname.unicode_normalize(:nfc)

      # 空白文字をトリム
      fname = fname.strip

      # 無効な文字をアンダースコアに置換
      fname = fname.gsub(/[<>:"|?*\x00-\x1f]/, "_")

      # パストラバーサルを防ぐためにパス区切り文字をアンダースコアに置換
      fname = fname.gsub(/[\/\\]/, "_")

      # 連続する.を_に置換（パストラバーサル対策）
      fname = fname.gsub(/\.{2,}/, "_")

      # 連続するアンダースコアを1つに縮める
      fname = fname.squeeze("_")

      # ファイル名がピリオドで始まる場合は削除（隠しファイルを防ぐ）
      fname = fname.sub(/^\.*/, "")

      # ファイル名と拡張子を分離
      basename = File.basename(fname, ".*")
      extension = File.extname(fname)

      # ベース名が空の場合はデフォルト名を使用
      basename = "file" if basename.empty?

      # Windowsの予約名をチェックして修正
      reserved_names = %w[
        CON PRN AUX NUL COM1 COM2 COM3 COM4 COM5 COM6 COM7 COM8 COM9
        LPT1 LPT2 LPT3 LPT4 LPT5 LPT6 LPT7 LPT8 LPT9
      ]

      if reserved_names.include?(basename.upcase)
        basename = "#{basename}_file"
      end

      # ファイル名を再構築
      fname = "#{basename}#{extension}"

      # ファイル名の長さ制限（255バイト）を適用
      if fname.bytesize > MAX_FILENAME_BYTES
        # 拡張子を保持しつつ、ファイル名を切り詰め
        max_basename_bytes = MAX_FILENAME_BYTES - extension.bytesize
        if max_basename_bytes > 0
          # UTF-8を考慮してバイト数で切り詰め
          basename = basename.byteslice(0, max_basename_bytes)
          # 不完全なUTF-8文字を削除
          basename = basename.not_nil!.scrub("")
          fname = "#{basename}#{extension}"
        else
          # 拡張子が長すぎる場合はファイル名全体を切り詰め
          fname = fname.byteslice(0, MAX_FILENAME_BYTES).not_nil!.scrub("")
        end
      end

      self.filename = fname
    end

    # ファイル名のバリデーション
    sig { void }
    private def validate_filename_format
      return if filename.blank?

      # この時点ではサニタイズ済みなので、
      # ファイル名が空になっていないかチェック
      if filename.not_nil!.strip.empty?
        errors.add(:filename, :blank)
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
