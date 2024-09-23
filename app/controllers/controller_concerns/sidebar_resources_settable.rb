# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module SidebarResourcesSettable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { void }
    private def set_joined_lists
      @joined_lists = T.let(viewer!.lists, T.nilable(List::PrivateRelation))
    end
  end
end
