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
      page_record = space_record.page_records.find_by!(number: params[:page_number])
      topic_record = page_record.topic_record.not_nil!
      topic_policy = topic_policy_for(topic_record:)

      unless topic_policy.can_create_edit_suggestion?
        return render_404
      end

      space_member_record = current_user_record!.space_member_record(space_record:)

      form = EditSuggestions::CreateForm.new(form_params)

      if form.invalid?
        page = PageRepository.new.to_model(page_record:, current_space_member: space_member_record)

        return render_component(
          EditSuggestions::NewView.new(form:, page:),
          status: :unprocessable_entity
        )
      end

      result = EditSuggestions::CreateService.new.call(
        space_member_record: space_member_record.not_nil!,
        page_record:,
        topic_record:,
        title: form.title.not_nil!,
        description: form.description.not_nil!,
        page_title: form.page_title.not_nil!,
        page_body: form.page_body.not_nil!
      )
      result.edit_suggestion_record

      flash[:notice] = t("messages.edit_suggestions.created")
      # TODO: ShowControllerが実装されたらedit_suggestion_pathに変更する
      redirect_to topic_edit_suggestion_list_path(
        space_identifier: space_record.identifier,
        topic_number: topic_record.number
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
