# typed: strict
# frozen_string_literal: true

module Navbars
  class TopComponent < ApplicationComponent
    use_helpers :signed_in?
  end
end
