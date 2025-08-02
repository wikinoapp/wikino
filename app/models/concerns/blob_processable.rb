# typed: strict
# frozen_string_literal: true

require "vips"

module BlobProcessable
  extend ::ActiveSupport::Concern
  extend T::Sig
  extend T::Helpers

  requires_ancestor { ActiveStorage::Blob }

  # 処理対象の画像形式
  SUPPORTED_IMAGE_FORMATS = T.let(
    %w[jpeg jpg png gif webp].freeze,
    T::Array[String]
  )

  sig { returns(T::Boolean) }
  def process_image_with_exif_removal
    return false unless supported_image_format?

    # ファイルをダウンロード
    tempfile = download_to_tempfile

    begin
      # EXIF情報を削除して自動回転を適用
      process_image(tempfile.path.not_nil!)

      # 処理済みファイルをアップロード
      upload_processed_file(tempfile.path.not_nil!)

      true
    rescue => e
      Rails.logger.error("Image processing failed: #{e.message}")
      false
    ensure
      tempfile.close
      tempfile.unlink
    end
  end

  private

  sig { returns(T::Boolean) }
  private def supported_image_format?
    # ActiveStorage::Blobのメソッドを明示的にキャスト
    blob = T.cast(self, ActiveStorage::Blob)
    extension = blob.filename.extension_without_delimiter.downcase
    SUPPORTED_IMAGE_FORMATS.include?(extension)
  end

  sig { returns(Tempfile) }
  private def download_to_tempfile
    # ActiveStorage::Blobのメソッドを明示的にキャスト
    blob = T.cast(self, ActiveStorage::Blob)
    tempfile = Tempfile.new(["image_processing", ".#{blob.filename.extension}"])
    tempfile.binmode
    tempfile.write(blob.download)
    tempfile.rewind
    tempfile
  end

  sig { params(input_path: String).void }
  private def process_image(input_path)
    # Vipsを使用して画像を処理
    image = Vips::Image.new_from_file(input_path)

    # EXIF情報に基づいて自動回転を適用
    image = image.autorot

    # EXIF情報を削除して保存
    # stripオプションでメタデータを削除
    blob = T.cast(self, ActiveStorage::Blob)
    case blob.filename.extension_without_delimiter.downcase
    when "jpg", "jpeg"
      image.jpegsave(input_path, strip: true, Q: 90)
    when "png"
      image.pngsave(input_path, strip: true, compression: 9)
    when "webp"
      image.webpsave(input_path, strip: true, Q: 90)
    else
      # GIFなどのその他の形式はそのまま保存
      image.write_to_file(input_path)
    end
  end

  sig { params(processed_path: String).void }
  private def upload_processed_file(processed_path)
    # 処理済みファイルを読み込んでActive Storageに再アップロード
    blob = T.cast(self, ActiveStorage::Blob)
    File.open(processed_path, "rb") do |file|
      blob.upload(file)
    end
  end
end
