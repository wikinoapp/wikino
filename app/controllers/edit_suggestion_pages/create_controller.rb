# typed: strict
# frozen_string_literal: true

module EditSuggestionPages
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicAware

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      topic_record = space_record.topic_records.find_by!(number: params[:topic_number])
      topic_policy = topic_policy_for(topic_record:)

      unless topic_policy.can_create_edit_suggestion?
        return render_404
      end

      space_member_record = current_user_record!.space_member_record(space_record:)

      # フォームから編集提案IDを取得
      edit_suggestion_id = form_params[:edit_suggestion_id]

      edit_suggestion_record = EditSuggestionRecord
        .open_or_draft
        .where(topic_id: topic_record.id, created_space_member_id: space_member_record.not_nil!.id)
        .find_by(id: edit_suggestion_id)

      # 編集提案が存在しない、または権限がない場合
      unless edit_suggestion_record
        return render_404
      end

      # ページ番号からページレコードを取得
      page_record = space_record.page_records.find_by!(number: params[:page_number])

      form = EditSuggestionPages::CreateForm.new(
        form_params.merge(
          space_member_record:,
          page_record:,
          edit_suggestion_record:
        )
      )

      if form.invalid?
        page = PageRepository.new.to_model(page_record:, current_space_member: space_member_record)

        # 既存の編集提案を取得（フォームで選択可能な編集提案）
        existing_edit_suggestion_records = EditSuggestionRecord
          .open_or_draft
          .where(topic_id: topic_record.id, created_space_member_id: space_member_record.not_nil!.id)
          .preload(:created_space_member_record)

        existing_edit_suggestions = EditSuggestionRepository.new.to_models(
          edit_suggestion_records: existing_edit_suggestion_records
        )

        return render_component(
          EditSuggestionPages::NewView.new(form:, page:, existing_edit_suggestions:),
          status: :unprocessable_entity
        )
      end

      # 既存の編集提案にページを追加
      EditSuggestionPages::AddService.new.call(
        edit_suggestion_record:,
        space_member_record: space_member_record.not_nil!,
        page_record:,
        page_title: form.page_title.not_nil!,
        page_body: form.page_body.not_nil!
      )

      flash[:notice] = t("messages.edit_suggestion_pages.added")
      # TODO: ShowControllerが実装されたらedit_suggestion_pathに変更する
      redirect_to topic_edit_suggestion_list_path(
        space_identifier: space_record.identifier,
        topic_number: topic_record.number
      )
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:edit_suggestion_pages_create_form), ActionController::Parameters).permit(
        :edit_suggestion_id,
        :page_title,
        :page_body
      )
    end
  end
end
