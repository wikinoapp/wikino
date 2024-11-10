# typed: true
# frozen_string_literal: true

module Atom
  class ShowController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :set_current_space

    sig { returns(T.untyped) }
    def call
      @pages = Current.space!.pages.published
        .joins(:topic).merge(Topic.visibility_public)
        .order(published_at: :desc, id: :desc)
        .limit(15)

      render(formats: :atom)
    end
  end
end
