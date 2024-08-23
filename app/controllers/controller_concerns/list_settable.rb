# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module ListSettable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { void }
    private def set_list
      @list = viewer!.space.lists.kept.find_by!(number: params[:list_number])
      authorize(@list, :show?)
    end
  end
end
