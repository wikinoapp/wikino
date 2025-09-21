# typed: true
# frozen_string_literal: true

module Pages
  class EditController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicAware

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      page_record = space_record.page_record_by_number!(params[:page_number])
      topic_policy = topic_policy_for(topic_record: page_record.topic_record.not_nil!)

      unless topic_policy.can_update_page?(page_record:)
        return render_404
      end

      space_member_record = current_user_record!.space_member_record(space_record:)
      draft_page_record = space_member_record.not_nil!.draft_page_records.find_by(page_record:)
      pageable_record = draft_page_record.presence || page_record

      space = SpaceRepository.new.to_model(space_record:)
      page = PageRepository.new.to_model(page_record:, current_space_member: space_member_record)

      form = Pages::EditForm.new(
        space_member_record:,
        topic_number: pageable_record.topic_record.not_nil!.number,
        title: pageable_record.title,
        body: pageable_record.body
      )

      link_list = LinkListRepository.new.to_model(user_record: current_user_record, pageable_record:)
      backlink_list = BacklinkListRepository.new.to_model(user_record: current_user_record, page_record:)
      current_user = UserRepository.new.to_model(user_record: current_user_record!)

      # 編集提案の作成が可能かチェック
      topic = TopicRepository.new.to_model(topic_record: page_record.topic_record.not_nil!)
      can_create_edit_suggestion = topic_policy.can_create_edit_suggestion?

      # 既存の編集提案（自分が作成した下書き/オープンのもの）を取得
      existing_edit_suggestions = if can_create_edit_suggestion && space_member_record
        edit_suggestion_records = EditSuggestionRecord
          .open_or_draft
          .where(
            topic_id: page_record.topic_record.not_nil!.id,
            created_space_member_id: space_member_record.not_nil!.id
          )
          .order(created_at: :desc)

        EditSuggestionRepository.new.to_models(edit_suggestion_records:)
      else
        []
      end

      render_component Pages::EditView.new(
        space:,
        page:,
        draft_page: draft_page(draft_page_record:),
        form:,
        link_list:,
        backlink_list:,
        current_user:,
        topic:,
        can_create_edit_suggestion:,
        existing_edit_suggestions:
      )
    end

    sig { params(draft_page_record: T.nilable(DraftPageRecord)).returns(T.nilable(DraftPage)) }
    private def draft_page(draft_page_record:)
      return if draft_page_record.nil?

      DraftPageRepository.new.to_model(draft_page_record: draft_page_record.not_nil!)
    end
  end
end
