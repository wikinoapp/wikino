# typed: true
# frozen_string_literal: true

require "marcel"

class AttachmentValidationService
  extend T::Sig

  # 許可するMIMEタイプのホワイトリスト
  ALLOWED_MIME_TYPES = T.let(
    Set.new([
      # 画像
      "image/jpeg",
      "image/jpg",
      "image/png",
      "image/gif",
      "image/webp",
      "image/svg+xml",
      # ドキュメント
      "application/pdf",
      "text/plain",
      "text/csv",
      # 圧縮ファイル
      "application/zip",
      "application/x-zip-compressed",
      # その他
      "application/json",
      "application/xml",
      "text/xml"
    ]).freeze,
    T::Set[String]
  )

  # 最大ファイルサイズ（50MB）
  MAX_FILE_SIZE = T.let(50.megabytes, Integer)

  sig { params(blob: ActiveStorage::Blob).returns(T::Boolean) }
  def self.valid?(blob)
    new(blob).valid?
  end

  sig { params(blob: ActiveStorage::Blob).void }
  def initialize(blob)
    @blob = T.let(blob, ActiveStorage::Blob)
  end

  sig { returns(T::Boolean) }
  def valid?
    valid_size? && valid_mime_type? && valid_content?
  end

  sig { returns(T::Array[String]) }
  def errors
    errors = []
    errors << "ファイルサイズが大きすぎます（最大#{MAX_FILE_SIZE / 1.megabyte}MB）" unless valid_size?
    errors << "許可されていないファイル形式です" unless valid_mime_type?
    errors << "ファイルの内容が不正です" unless valid_content?
    errors
  end

  private

  sig { returns(T::Boolean) }
  private def valid_size?
    @blob.byte_size <= MAX_FILE_SIZE
  end

  sig { returns(T::Boolean) }
  private def valid_mime_type?
    ALLOWED_MIME_TYPES.include?(@blob.content_type || "")
  end

  sig { returns(T::Boolean) }
  private def valid_content?
    # ファイルの実際の内容を検証（マジックナンバーチェック）
    return false if @blob.download.blank?

    # Marcelを使用してファイルの実際のMIMEタイプを検出
    detected_mime_type = Marcel::MimeType.for(
      @blob.download,
      name: @blob.filename.to_s
    )

    # 検出されたMIMEタイプが許可リストに含まれているか確認
    ALLOWED_MIME_TYPES.include?(detected_mime_type)
  rescue
    # ダウンロードエラーやその他のエラーが発生した場合は無効とする
    false
  end
end

