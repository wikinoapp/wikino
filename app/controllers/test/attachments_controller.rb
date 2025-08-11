# typed: true
# frozen_string_literal: true

module Test
  class AttachmentsController < ApplicationController
    # テスト環境でのみ使用されるモックエンドポイント
    # ActiveStorageを迂回してファイルアップロードのテストを可能にする

    def presign
      # テスト環境でのみ動作
      unless Rails.env.test?
        raise ActionController::RoutingError, "Not Found"
      end

      # プリサインURLのモックレスポンス
      render json: {
        directUploadUrl: "https://test.example.com/upload",
        directUploadHeaders: {"Content-Type" => params[:content_type] || "image/png"},
        blobSignedId: "test-blob-#{SecureRandom.hex(8)}",
        attachmentId: "test-attachment-#{SecureRandom.hex(8)}"
      }
    end

    def upload
      # テスト環境でのみ動作
      unless Rails.env.test?
        raise ActionController::RoutingError, "Not Found"
      end

      # アップロード成功をシミュレート
      head :ok
    end
  end
end
