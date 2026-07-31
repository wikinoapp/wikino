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

    # ダウンロードに失敗した場合はfalseを返す
    return false if tempfile.nil?

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
      if tempfile
        tempfile.close
        tempfile.unlink
      end
    end
  end

  sig { returns(T::Boolean) }
  private def supported_image_format?
    # ActiveStorage::Blobのメソッドを明示的にキャスト
    blob = T.cast(self, ActiveStorage::Blob)
    extension = blob.filename.extension_without_delimiter.downcase
    SUPPORTED_IMAGE_FORMATS.include?(extension)
  end

  sig { returns(T.nilable(Tempfile)) }
  private def download_to_tempfile
    # ActiveStorage::Blobのメソッドを明示的にキャスト
    blob = T.cast(self, ActiveStorage::Blob)
    tempfile = Tempfile.new(["image_processing", ".#{blob.filename.extension}"])
    tempfile.binmode

    begin
      tempfile.write(blob.download)
    rescue Aws::S3::Errors::NoSuchKey, ActiveStorage::FileNotFoundError => e
      # S3にファイルが見つからない場合はnilを返す
      Rails.logger.warn("Blob download failed: #{e.message} (blob_id: #{blob.id})")
      tempfile.close
      tempfile.unlink
      return nil
    end

    tempfile.rewind
    tempfile
  end

  sig { params(input_path: String).void }
  private def process_image(input_path)
    blob = T.cast(self, ActiveStorage::Blob)
    extension = blob.filename.extension_without_delimiter.downcase

    # GIFファイルは処理をスキップ（アニメーションを保持）
    return if extension == "gif"

    # Vipsを使用して画像を処理
    image = Vips::Image.new_from_file(input_path)

    # EXIF情報に基づいて自動回転を適用
    image = image.autorot

    # libvips evaluates lazily and keeps the input file mapped until the save
    # finishes, so saving over the path being read from truncates data that is
    # still in use. Write to a separate temporary file and swap it in once the
    # save is done. The temporary file has to carry the same extension as the
    # input because `write_to_file` derives the output format from the path.
    #
    # [Ja] libvips は遅延評価で、保存が終わるまで入力ファイルをマップしたまま
    # 保持する。そのため読み込み元と同じパスへ保存すると、まだ使用中のデータを
    # 切り詰めてしまう。別の一時ファイルへ書き出し、保存後に読み込み元へ
    # 差し替える。`write_to_file` は出力形式をパスの拡張子から決めるため、
    # 一時ファイルの拡張子は読み込み元と揃える必要がある。
    Tempfile.create(["image_processing", File.extname(input_path)]) do |output|
      # libvips writes through the path it is given, so the open handle is not
      # needed.
      #
      # [Ja] libvips は渡されたパス経由で書き出すため、開いたハンドルは使わない。
      output.close

      case extension
      when "jpg", "jpeg"
        image.jpegsave(output.path, strip: true, Q: 90)
      when "png"
        image.pngsave(output.path, strip: true, compression: 9)
      when "webp"
        image.webpsave(output.path, strip: true, Q: 90)
      else
        image.write_to_file(output.path)
      end

      FileUtils.mv(output.path, input_path)
    end
  end

  sig { params(processed_path: String).void }
  private def upload_processed_file(processed_path)
    # 処理済みファイルを読み込んでActive Storageに再アップロード
    blob = T.cast(self, ActiveStorage::Blob)
    File.open(processed_path, "rb") do |file|
      # ファイルサイズとチェックサムを更新
      blob.byte_size = file.size
      blob.checksum = Digest::MD5.base64digest(file.read)
      file.rewind

      # ファイルをアップロード
      blob.upload(file)

      # メタデータを保存
      blob.save!
    end
  end
end
