# typed: true
# frozen_string_literal: true

module Pages
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!
      space_viewer = Current.viewer!.space_viewer!(space:)
      topic = space.topics.kept.find_by!(number: params[:topic_number])

      unless space_viewer.can_create_page?(topic:)
        return render_404
      end

      result = CreateBlankedPageUseCase.new.call(topic:)

      redirect_to edit_page_path(space.identifier, result.page.number)
    end
  end
end
