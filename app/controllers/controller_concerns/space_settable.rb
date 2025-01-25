# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module SpaceSettable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { void }
    def set_current_space
      @space = Space.kept.find_by!(identifier: params[:space_identifier])
    end
  end
end
