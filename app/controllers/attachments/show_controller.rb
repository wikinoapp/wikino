# typed: strict
# frozen_string_literal: true

module Attachments
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable

    before_action :restore_user_session

    # GET /attachments/:id
    # NOTE: このエンドポイントは添付ファイルを作ったときページ本文に挿入される
    #       このエンドポイントを `/s/:space_identifier/` 配下に置いていないのは、
    #       スペースの識別子を変えたときにページ本文内の添付ファイルのURLが無効になるのを防ぐため
    sig { void }
    def call
      attachment_record = AttachmentRecord.find(params[:attachment_id])

      # アクセス権限をチェック
      unless attachment_record.viewable_by?(user_record: current_user_record)
        render_404
        return
      end

      # リダイレクト先のURLを取得
      redirect_url = attachment_record.redirect_url
      if redirect_url
        redirect_to(redirect_url, allow_other_host: true)
      else
        render_404
      end
    end
  end
end
