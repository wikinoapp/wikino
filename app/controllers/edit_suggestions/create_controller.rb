# typed: strict
# frozen_string_literal: true

module EditSuggestions
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

      # 編集提案の作成権限をチェック
      unless topic_policy.can_create_edit_suggestion?
        return render_404
      end

      space_member_record = current_user_record!.space_member_record(space_record:)
      page_record = params[:page_number].present? ? space_record.page_records.find_by(number: params[:page_number]) : nil

      form = EditSuggestions::CreateForm.new(form_params)

      if form.invalid?
        space = SpaceRepository.new.to_model(space_record:)
        topic = TopicRepository.new.to_model(topic_record:)
        page = page_record ? PageRepository.new.to_model(page_record:, current_space_member: space_member_record) : nil

        return render_component(
          EditSuggestions::NewView.new(
            form:,
            space:,
            topic:,
            page:
          ),
          status: :unprocessable_entity
        )
      end

      # 新規編集提案を作成（このコントローラーは新規作成専用）
      result = EditSuggestions::CreateService.new.call(
        space_member_record: space_member_record.not_nil!,
        page_record:,
        topic_record:,
        title: form.title.not_nil!,
        description: form.description.not_nil!,
        page_title: form.page_title.not_nil!,
        page_body: form.page_body.not_nil!
      )
      edit_suggestion_record = result.edit_suggestion_record

      flash[:notice] = t("messages.edit_suggestions.created")
      redirect_to edit_suggestion_path(
        space_identifier: space_record.identifier,
        topic_number: topic_record.number,
        id: edit_suggestion_record.id
      )
    end

    sig { returns(ActionController::Parameters) }
    private def form_params
      T.cast(params.require(:edit_suggestions_create_form), ActionController::Parameters).permit(
        :title,
        :description,
        :page_title,
        :page_body
      )
    end
  end
end
