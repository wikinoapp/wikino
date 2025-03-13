# typed: true
# frozen_string_literal: true

module Topics
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

      unless space_viewer.can_create_topic?
        return render_404
      end

      form = NewTopicForm.new

      render Topics::NewView.new(
        space_entity: space.to_entity(space_viewer:),
        form:
      )
    end
  end
end
