# typed: true
# frozen_string_literal: true

module Test
  module Attachments
    module Uploads
      class CreateController < ApplicationController
        # テスト環境でのみ使用されるモックエンドポイント
        # ActiveStorageを迂回してファイルアップロードのテストを可能にする

        def call
          # テスト環境でのみ動作
          unless Rails.env.test?
            raise ActionController::RoutingError, "Not Found"
          end

          # アップロード成功をシミュレート
          head :ok
        end
      end
    end
  end
end