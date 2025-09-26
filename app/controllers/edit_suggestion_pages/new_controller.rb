# typed: strict
# frozen_string_literal: true

module EditSuggestionPages
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicAware
    include ControllerConcerns::EditSuggestionRenderable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      page_record = space_record.page_record_by_number!(params[:page_number])
      topic_record = page_record.topic_record.not_nil!
      topic_policy = topic_policy_for(topic_record:)

      # 編集提案の作成権限をチェック
      unless topic_policy.can_create_edit_suggestion?
        return render_404
      end

      space_member_record = current_user_record!.space_member_record(space_record:)

      # ページの下書きまたは公開版からデータを取得
      draft_page_record = space_member_record.not_nil!.draft_page_records.find_by(page_record:)
      pageable_record = draft_page_record.presence || page_record

      form = EditSuggestionPages::CreateForm.new(
        page_title: pageable_record.title,
        page_body: pageable_record.body
      )

      view = build_edit_suggestion_page_view(
        form:,
        page_record:,
        space_member_record:
      )

      render(view, layout: false)
    end
  end
end
