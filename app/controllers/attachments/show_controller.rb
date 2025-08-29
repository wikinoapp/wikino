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
      space_record = attachment_record.space_record.not_nil!
      space_member_record = current_user_record&.space_member_record(space_record:)
      policy = SpaceMemberPolicyFactory.build(user_record: current_user_record, space_member_record:)

      unless policy.can_view_attachment?(attachment_record:)
        render_404
        return
      end

      # リダイレクト先のURLを取得
      redirect_url = attachment_record.redirect_url

      unless redirect_url
        render_404
        return
      end

      redirect_to(redirect_url, allow_other_host: true)
    end
  end
end
