# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe BlobProcessable do
  describe "#process_image_with_exif_removal" do
    it "サポートされている画像形式の場合、処理が成功するとtrueを返す" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      tempfile = Tempfile.new(["test", ".jpg"])
      file_content = "fake image content"

      allow(blob).to receive_messages(
        filename: ActiveStorage::Filename.new("test.jpg"),
        download: file_content,
        download_to_tempfile: tempfile,
        process_image: nil,
        upload_processed_file: nil
      )

      expect(blob.process_image_with_exif_removal).to be true

      tempfile&.close unless tempfile&.closed?
      tempfile&.unlink if tempfile&.path && File.exist?(tempfile.path)
    end

    it "サポートされている画像形式の場合、画像処理とアップロードを実行する" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      tempfile = Tempfile.new(["test", ".jpg"])
      file_content = "fake image content"
      tempfile_path = tempfile.path

      allow(blob).to receive_messages(
        filename: ActiveStorage::Filename.new("test.jpg"),
        download: file_content,
        download_to_tempfile: tempfile
      )

      allow(blob).to receive(:process_image)
      allow(blob).to receive(:upload_processed_file)

      blob.process_image_with_exif_removal

      expect(blob).to have_received(:process_image).with(tempfile_path)
      expect(blob).to have_received(:upload_processed_file).with(tempfile_path)

      tempfile&.close unless tempfile&.closed?
      tempfile&.unlink if tempfile&.path && File.exist?(tempfile.path)
    end

    it "処理が成功した場合、tempfileをクリーンアップする" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      tempfile = Tempfile.new(["test", ".jpg"])
      file_content = "fake image content"

      allow(blob).to receive_messages(
        filename: ActiveStorage::Filename.new("test.jpg"),
        download: file_content,
        download_to_tempfile: tempfile,
        process_image: nil,
        upload_processed_file: nil
      )

      blob.process_image_with_exif_removal
      expect(File.exist?(tempfile.path)).to be false if tempfile.path
    end

    it "処理中にエラーが発生する場合、falseを返す" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      tempfile = Tempfile.new(["test", ".jpg"])

      allow(blob).to receive_messages(
        filename: ActiveStorage::Filename.new("test.jpg"),
        download: "fake content",
        download_to_tempfile: tempfile
      )
      allow(blob).to receive(:process_image).and_raise(StandardError, "Processing error")

      expect(blob.process_image_with_exif_removal).to be false

      tempfile&.close unless tempfile&.closed?
      tempfile&.unlink if tempfile&.path && File.exist?(tempfile.path)
    end

    it "処理中にエラーが発生する場合、エラーをログに記録する" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      tempfile = Tempfile.new(["test", ".jpg"])

      allow(blob).to receive_messages(
        filename: ActiveStorage::Filename.new("test.jpg"),
        download: "fake content",
        download_to_tempfile: tempfile
      )
      allow(blob).to receive(:process_image).and_raise(StandardError, "Processing error")

      allow(Rails.logger).to receive(:error)
      blob.process_image_with_exif_removal
      expect(Rails.logger).to have_received(:error).with("Image processing failed: Processing error")

      tempfile&.close unless tempfile&.closed?
      tempfile&.unlink if tempfile&.path && File.exist?(tempfile.path)
    end

    it "処理中にエラーが発生する場合でも、tempfileをクリーンアップする" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      tempfile = Tempfile.new(["test", ".jpg"])

      allow(blob).to receive_messages(
        filename: ActiveStorage::Filename.new("test.jpg"),
        download: "fake content",
        download_to_tempfile: tempfile
      )
      allow(blob).to receive(:process_image).and_raise(StandardError, "Processing error")

      blob.process_image_with_exif_removal
      expect(File.exist?(tempfile.path)).to be false if tempfile.path
    end

    it "サポートされていない画像形式の場合、falseを返す" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new

      allow(blob).to receive(:filename).and_return(ActiveStorage::Filename.new("test.txt"))

      expect(blob.process_image_with_exif_removal).to be false
    end

    it "サポートされていない画像形式の場合、処理を実行しない" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new

      allow(blob).to receive(:filename).and_return(ActiveStorage::Filename.new("test.txt"))
      allow(blob).to receive(:download)

      blob.process_image_with_exif_removal

      expect(blob).not_to have_received(:download)
    end
  end

  describe "#supported_image_format?" do
    %w[jpeg jpg png gif webp].each do |format|
      it "#{format}ファイルの場合、trueを返す" do
        blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
        blob = blob_class.new

        allow(blob).to receive(:filename).and_return(ActiveStorage::Filename.new("test.#{format}"))
        expect(blob.send(:supported_image_format?)).to be true
      end

      it "#{format.upcase}ファイル（大文字）の場合、trueを返す" do
        blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
        blob = blob_class.new

        allow(blob).to receive(:filename).and_return(ActiveStorage::Filename.new("test.#{format.upcase}"))
        expect(blob.send(:supported_image_format?)).to be true
      end
    end

    it "サポートされていない形式の場合、falseを返す" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new

      allow(blob).to receive(:filename).and_return(ActiveStorage::Filename.new("test.txt"))
      expect(blob.send(:supported_image_format?)).to be false
    end
  end

  describe "#process_image" do
    it "GIFファイルの場合、処理をスキップする（アニメーションを保持）" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      input_path = "/tmp/test_image.gif"

      allow(blob).to receive(:filename).and_return(ActiveStorage::Filename.new("animated.gif"))

      allow(Vips::Image).to receive(:new_from_file)
      blob.send(:process_image, input_path)
      expect(Vips::Image).not_to have_received(:new_from_file)
    end

    it "JPEGファイルの場合、EXIF情報を削除して保存する" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      input_path = "/tmp/test_image.jpg"
      vips_image = double(:vips_image)

      allow(blob).to receive(:filename).and_return(ActiveStorage::Filename.new("test.jpg"))
      allow(Vips::Image).to receive(:new_from_file).with(input_path).and_return(vips_image)
      allow(vips_image).to receive(:autorot).and_return(vips_image)
      allow(vips_image).to receive(:jpegsave)

      blob.send(:process_image, input_path)

      expect(vips_image).to have_received(:autorot)
      expect(vips_image).to have_received(:jpegsave).with(input_path, strip: true, Q: 90)
    end

    it "PNGファイルの場合、メタデータを削除して保存する" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      input_path = "/tmp/test_image.png"
      vips_image = double(:vips_image)

      allow(blob).to receive(:filename).and_return(ActiveStorage::Filename.new("test.png"))
      allow(Vips::Image).to receive(:new_from_file).with(input_path).and_return(vips_image)
      allow(vips_image).to receive(:autorot).and_return(vips_image)
      allow(vips_image).to receive(:pngsave)

      blob.send(:process_image, input_path)

      expect(vips_image).to have_received(:autorot)
      expect(vips_image).to have_received(:pngsave).with(input_path, strip: true, compression: 9)
    end

    it "WebPファイルの場合、メタデータを削除して保存する" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      input_path = "/tmp/test_image.webp"
      vips_image = double(:vips_image)

      allow(blob).to receive(:filename).and_return(ActiveStorage::Filename.new("test.webp"))
      allow(Vips::Image).to receive(:new_from_file).with(input_path).and_return(vips_image)
      allow(vips_image).to receive(:autorot).and_return(vips_image)
      allow(vips_image).to receive(:webpsave)

      blob.send(:process_image, input_path)

      expect(vips_image).to have_received(:autorot)
      expect(vips_image).to have_received(:webpsave).with(input_path, strip: true, Q: 90)
    end

    it "その他の形式の場合、そのまま保存する" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      input_path = "/tmp/test_image.bmp"
      vips_image = double(:vips_image)

      allow(blob).to receive(:filename).and_return(ActiveStorage::Filename.new("test.bmp"))
      allow(Vips::Image).to receive(:new_from_file).with(input_path).and_return(vips_image)
      allow(vips_image).to receive(:autorot).and_return(vips_image)
      allow(vips_image).to receive(:write_to_file)

      blob.send(:process_image, input_path)

      expect(vips_image).to have_received(:autorot)
      expect(vips_image).to have_received(:write_to_file).with(input_path)
    end
  end

  describe "#download_to_tempfile" do
    it "tempfileを作成してファイル内容を書き込む" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      file_content = "fake image binary content"

      allow(blob).to receive_messages(
        filename: ActiveStorage::Filename.new("test.jpg"),
        download: file_content
      )

      tempfile = blob.send(:download_to_tempfile)

      expect(tempfile).to be_a(Tempfile)
      expect(tempfile.path).to include("image_processing")
      expect(tempfile.path).to include(".jpg")

      tempfile.rewind
      expect(tempfile.read).to eq(file_content)

      tempfile.close
      tempfile.unlink
    end
  end

  describe "#upload_processed_file" do
    it "処理済みファイルをアップロードする" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      processed_path = "/tmp/processed_image.jpg"
      file_content = "processed image content"

      allow(File).to receive(:open).with(processed_path, "rb").and_yield(StringIO.new(file_content))

      allow(blob).to receive(:upload)
      blob.send(:upload_processed_file, processed_path)
      expect(blob).to have_received(:upload).with(kind_of(StringIO))
    end
  end

  describe "統合テスト: アニメーションGIFの処理" do
    it "アニメーションGIFをアップロードした場合、アニメーション情報を保持したまま処理を完了する" do
      blob_class = Class.new(ActiveStorage::Blob) { include BlobProcessable }
      blob = blob_class.new
      gif_content = "GIF89a animated gif content with multiple frames"
      tempfile = Tempfile.new(["animated", ".gif"])

      allow(blob).to receive_messages(
        filename: ActiveStorage::Filename.new("animated.gif"),
        download: gif_content,
        download_to_tempfile: tempfile
      )
      allow(File).to receive(:open).with(tempfile.path, "rb").and_yield(StringIO.new(gif_content))

      # Vipsによる画像処理が呼ばれないことを確認
      allow(Vips::Image).to receive(:new_from_file)

      # アップロードは実行される
      allow(blob).to receive(:upload)

      result = blob.process_image_with_exif_removal
      expect(result).to be true
      expect(Vips::Image).not_to have_received(:new_from_file)
      expect(blob).to have_received(:upload).with(kind_of(StringIO))

      tempfile&.close unless tempfile&.closed?
      tempfile&.unlink if tempfile&.path && File.exist?(tempfile.path)
    end
  end
end
