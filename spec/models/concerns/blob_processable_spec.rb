# typed: false
# frozen_string_literal: true

require "rails_helper"

RSpec.describe BlobProcessable do
  # ActiveStorage::Blobを拡張したテスト用クラス
  let(:blob_class) do
    Class.new(ActiveStorage::Blob) do
      include BlobProcessable
    end
  end

  let(:blob) { blob_class.new }

  describe "#process_image_with_exif_removal" do
    context "サポートされている画像形式の場合" do
      before do
        allow(blob).to receive(:filename).and_return(
          ActiveStorage::Filename.new("test.jpg")
        )
      end

      context "処理が成功する場合" do
        let(:tempfile) { Tempfile.new(["test", ".jpg"]) }
        let(:file_content) { "fake image content" }

        before do
          allow(blob).to receive(:download).and_return(file_content)
          allow(blob).to receive(:download_to_tempfile).and_return(tempfile)
          allow(blob).to receive(:process_image)
          allow(blob).to receive(:upload_processed_file)
        end

        after do
          tempfile.close if tempfile && !tempfile.closed?
          tempfile.unlink if tempfile && tempfile.path && File.exist?(tempfile.path)
        end

        it "trueを返す" do
          expect(blob.process_image_with_exif_removal).to be true
        end

        it "画像処理とアップロードを実行する" do
          expect(blob).to receive(:process_image).with(tempfile.path)
          expect(blob).to receive(:upload_processed_file).with(tempfile.path)
          blob.process_image_with_exif_removal
        end

        it "tempfileをクリーンアップする" do
          blob.process_image_with_exif_removal
          expect(File.exist?(tempfile.path)).to be false if tempfile.path
        end
      end

      context "処理中にエラーが発生する場合" do
        let(:tempfile) { Tempfile.new(["test", ".jpg"]) }

        before do
          allow(blob).to receive(:download).and_return("fake content")
          allow(blob).to receive(:download_to_tempfile).and_return(tempfile)
          allow(blob).to receive(:process_image).and_raise(StandardError, "Processing error")
        end

        after do
          tempfile.close if tempfile && !tempfile.closed?
          tempfile.unlink if tempfile && tempfile.path && File.exist?(tempfile.path)
        end

        it "falseを返す" do
          expect(blob.process_image_with_exif_removal).to be false
        end

        it "エラーをログに記録する" do
          expect(Rails.logger).to receive(:error).with("Image processing failed: Processing error")
          blob.process_image_with_exif_removal
        end

        it "tempfileをクリーンアップする" do
          blob.process_image_with_exif_removal
          expect(File.exist?(tempfile.path)).to be false if tempfile.path
        end
      end
    end

    context "サポートされていない画像形式の場合" do
      before do
        allow(blob).to receive(:filename).and_return(
          ActiveStorage::Filename.new("test.txt")
        )
      end

      it "falseを返す" do
        expect(blob.process_image_with_exif_removal).to be false
      end

      it "処理を実行しない" do
        expect(blob).not_to receive(:download)
        blob.process_image_with_exif_removal
      end
    end
  end

  describe "#supported_image_format?" do
    BlobProcessable::SUPPORTED_IMAGE_FORMATS.each do |format|
      context "#{format}ファイルの場合" do
        before do
          allow(blob).to receive(:filename).and_return(
            ActiveStorage::Filename.new("test.#{format}")
          )
        end

        it "trueを返す" do
          expect(blob.send(:supported_image_format?)).to be true
        end
      end

      context "#{format.upcase}ファイルの場合（大文字）" do
        before do
          allow(blob).to receive(:filename).and_return(
            ActiveStorage::Filename.new("test.#{format.upcase}")
          )
        end

        it "trueを返す" do
          expect(blob.send(:supported_image_format?)).to be true
        end
      end
    end

    context "サポートされていない形式の場合" do
      before do
        allow(blob).to receive(:filename).and_return(
          ActiveStorage::Filename.new("test.txt")
        )
      end

      it "falseを返す" do
        expect(blob.send(:supported_image_format?)).to be false
      end
    end
  end

  describe "#process_image" do
    let(:input_path) { "/tmp/test_image.jpg" }
    let(:vips_image) { double("Vips::Image") }

    context "GIFファイルの場合" do
      before do
        allow(blob).to receive(:filename).and_return(
          ActiveStorage::Filename.new("animated.gif")
        )
      end

      it "処理をスキップする（アニメーションを保持）" do
        expect(Vips::Image).not_to receive(:new_from_file)
        blob.send(:process_image, input_path)
      end
    end

    context "JPEGファイルの場合" do
      before do
        allow(blob).to receive(:filename).and_return(
          ActiveStorage::Filename.new("test.jpg")
        )
        allow(Vips::Image).to receive(:new_from_file).with(input_path).and_return(vips_image)
        allow(vips_image).to receive(:autorot).and_return(vips_image)
        allow(vips_image).to receive(:jpegsave)
      end

      it "EXIF情報を削除して保存する" do
        expect(vips_image).to receive(:autorot)
        expect(vips_image).to receive(:jpegsave).with(input_path, strip: true, Q: 90)
        blob.send(:process_image, input_path)
      end
    end

    context "PNGファイルの場合" do
      before do
        allow(blob).to receive(:filename).and_return(
          ActiveStorage::Filename.new("test.png")
        )
        allow(Vips::Image).to receive(:new_from_file).with(input_path).and_return(vips_image)
        allow(vips_image).to receive(:autorot).and_return(vips_image)
        allow(vips_image).to receive(:pngsave)
      end

      it "メタデータを削除して保存する" do
        expect(vips_image).to receive(:autorot)
        expect(vips_image).to receive(:pngsave).with(input_path, strip: true, compression: 9)
        blob.send(:process_image, input_path)
      end
    end

    context "WebPファイルの場合" do
      before do
        allow(blob).to receive(:filename).and_return(
          ActiveStorage::Filename.new("test.webp")
        )
        allow(Vips::Image).to receive(:new_from_file).with(input_path).and_return(vips_image)
        allow(vips_image).to receive(:autorot).and_return(vips_image)
        allow(vips_image).to receive(:webpsave)
      end

      it "メタデータを削除して保存する" do
        expect(vips_image).to receive(:autorot)
        expect(vips_image).to receive(:webpsave).with(input_path, strip: true, Q: 90)
        blob.send(:process_image, input_path)
      end
    end

    context "その他の形式の場合" do
      before do
        allow(blob).to receive(:filename).and_return(
          ActiveStorage::Filename.new("test.bmp")
        )
        allow(Vips::Image).to receive(:new_from_file).with(input_path).and_return(vips_image)
        allow(vips_image).to receive(:autorot).and_return(vips_image)
        allow(vips_image).to receive(:write_to_file)
      end

      it "そのまま保存する" do
        expect(vips_image).to receive(:autorot)
        expect(vips_image).to receive(:write_to_file).with(input_path)
        blob.send(:process_image, input_path)
      end
    end
  end

  describe "#download_to_tempfile" do
    let(:file_content) { "fake image binary content" }

    before do
      allow(blob).to receive(:filename).and_return(
        ActiveStorage::Filename.new("test.jpg")
      )
      allow(blob).to receive(:download).and_return(file_content)
    end

    it "tempfileを作成してファイル内容を書き込む" do
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
    let(:processed_path) { "/tmp/processed_image.jpg" }
    let(:file_content) { "processed image content" }

    before do
      allow(File).to receive(:open).with(processed_path, "rb").and_yield(StringIO.new(file_content))
    end

    it "処理済みファイルをアップロードする" do
      expect(blob).to receive(:upload).with(kind_of(StringIO))
      blob.send(:upload_processed_file, processed_path)
    end
  end

  describe "統合テスト: アニメーションGIFの処理" do
    context "アニメーションGIFをアップロードした場合" do
      let(:gif_content) { "GIF89a animated gif content with multiple frames" }
      let(:tempfile) { Tempfile.new(["animated", ".gif"]) }

      before do
        allow(blob).to receive(:filename).and_return(
          ActiveStorage::Filename.new("animated.gif")
        )
        allow(blob).to receive(:download).and_return(gif_content)
        allow(blob).to receive(:download_to_tempfile).and_return(tempfile)
        
        # GIFファイルは処理をスキップすることを確認
        allow(File).to receive(:open).with(tempfile.path, "rb").and_yield(StringIO.new(gif_content))
      end

      after do
        tempfile.close if tempfile && !tempfile.closed?
        tempfile.unlink if tempfile && tempfile.path && File.exist?(tempfile.path)
      end

      it "アニメーション情報を保持したまま処理を完了する" do
        # Vipsによる画像処理が呼ばれないことを確認
        expect(Vips::Image).not_to receive(:new_from_file)
        
        # アップロードは実行される
        expect(blob).to receive(:upload).with(kind_of(StringIO))
        
        result = blob.process_image_with_exif_removal
        expect(result).to be true
      end
    end
  end
end