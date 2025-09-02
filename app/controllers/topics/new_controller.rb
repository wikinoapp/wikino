# typed: true
# frozen_string_literal: true

module Topics
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceAware

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = current_space_record
      space_policy = space_policy_for(space_record:)

      unless space_policy.can_create_topic?
        return render_404
      end

      space = SpaceRepository.new.to_model(space_record:)
      form = Topics::CreationForm.new

      render_component Topics::NewView.new(current_user: current_user!, space:, form:)
    end
  end
end
