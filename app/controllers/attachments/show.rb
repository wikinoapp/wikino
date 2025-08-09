# typed: strict
# frozen_string_literal: true

module Attachments
  class Show < ApplicationController
    include ControllerConcerns::Authenticatable

    # GET /attachments/:id
    # NOTE: このエンドポイントは添付ファイルを作ったときページ本文に挿入される
    #       このエンドポイントを `/s/:space_identifier/` 配下に置いていないのは、
    #       スペースの識別子を変えたときにページ本文内の添付ファイルのURLが無効になるのを防ぐため
    sig { void }
    def call
      attachment_record = AttachmentRecord.find(params[:attachment_id])

      # page_attachment_references を経由して添付されているページを取得
      page_attachment_refs = PageAttachmentReferenceRecord
        .preload(:page_record)
        .where(attachment_record:)

      # アクセス権限をチェック
      # いずれかのページへのアクセス権限があればOK
      can_access = page_attachment_refs.any? do |ref|
        page_record = ref.page_record
        next false if page_record.nil?

        space_record = page_record.space_record
        next false if space_record.nil?

        # スペースメンバーかどうかをチェック
        if current_user_record
          SpaceMemberRecord.exists?(
            space_id: space_record.id,
            user_id: current_user_record.not_nil!.id
          )
        else
          # ログインしていない場合は公開スペースとみなす
          # TODO: スペースの公開設定を確認する必要がある場合は別途実装
          true
        end
      end

      unless can_access
        render_404
        return
      end

      # Active Storageの署名付きURLを生成してリダイレクト
      active_storage_attachment = attachment_record.active_storage_attachment_record
      blob = active_storage_attachment&.blob
      if blob
        # Active Storageのサービス署名付きURLを使用
        redirect_to(blob.url, allow_other_host: true)
      else
        render_404
      end
    end
  end
end
