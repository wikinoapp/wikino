# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module Deletions
      class NewView < ApplicationView
        extend T::Sig

        sig { returns(Space) }
        attr_reader :space

        sig { params(space: Space).void }
        def initialize(space:)
          @space = space
        end

        sig { returns(String) }
        def title
          "スペースの削除"
        end

        sig { returns(String) }
        def warning_title
          "警告"
        end

        sig { returns(String) }
        def warning_message
          "この操作は取り消せません。スペースとその中のすべてのトピック、ページが完全に削除されます。"
        end

        sig { returns(String) }
        def confirmation_message
          "削除を続行するには、スペース名「#{space.name}」を入力してください。"
        end

        sig { returns(String) }
        def space_name_label
          "スペース名"
        end

        sig { returns(String) }
        def cancel_button_text
          "キャンセル"
        end

        sig { returns(String) }
        def delete_button_text
          "スペースを削除"
        end
      end
    end
  end
end
