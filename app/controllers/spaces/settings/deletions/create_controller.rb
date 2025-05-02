# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Deletions
      class CreateController < ApplicationController
        extend T::Sig

        sig { void }
        def call
          @space = Space.find_by!(identifier: params[:space_identifier])

          if @space.name == params[:space_name]
            @space.destroy!
            redirect_to root_path, notice: "スペースを削除しました"
          else
            redirect_to space_settings_new_deletion_path(@space), alert: "スペース名が正しくありません"
          end
        end
      end
    end
  end
end
