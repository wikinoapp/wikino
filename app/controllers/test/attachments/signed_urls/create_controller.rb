# typed: true
# frozen_string_literal: true

module Test
  module Attachments
    module SignedUrls
      class CreateController < ApplicationController
        # テスト環境でのみ使用されるモックエンドポイント
        # 添付ファイルの署名付きURLのテストを可能にする

        def call
          # テスト環境でのみ動作
          unless Rails.env.test?
            raise ActionController::RoutingError, "Not Found"
          end

          attachment_ids = params.fetch(:attachment_ids, [])

          # テスト用の署名付きURLを生成
          signed_urls = {}
          attachment_ids.each do |id|
            # 1x1の透明なPNG画像のdata URI
            signed_urls[id] = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
          end

          render json: {signed_urls:}, status: :ok
        end
      end
    end
  end
end
