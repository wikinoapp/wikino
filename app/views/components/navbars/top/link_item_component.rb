# typed: strict
# frozen_string_literal: true

module Navbars
  module Top
    class LinkItemComponent < ApplicationComponent
      sig { params(current_page_name: PageName, page_name: PageName, class_name: String).void }
      def initialize(current_page_name:, page_name:, class_name: "")
        @current_page_name = current_page_name
        @page_name = page_name
        @class_name = class_name
      end

      sig { returns(PageName) }
      attr_reader :current_page_name
      private :current_page_name

      sig { returns(PageName) }
      attr_reader :page_name
      private :page_name

      sig { returns(String) }
      attr_reader :class_name
      private :class_name

      sig { returns(String) }
      def path
        case page_name
        when PageName::Home
          home_path
        when PageName::Inbox
          "#"
        when PageName::SignIn
          sign_in_path
        else
          T.absurd(page_name)
        end
      end

      sig { returns(String) }
      def title
        case page_name
        when PageName::Home
          t("nouns.home")
        when PageName::Inbox
          t("nouns.inbox")
        when PageName::SignIn
          t("nouns.sign_in")
        else
          T.absurd(page_name)
        end
      end

      def icon_name
        case page_name
        when PageName::Home
          "house"
        when PageName::Inbox
          "tray"
        when PageName::SignIn
          "sign-in"
        else
          T.absurd(page_name)
        end
      end

      sig { returns(String) }
      def icon_name_with_suffix
        suffix = (current_page_name == page_name) ? "-fill" : "-regular"

        "#{icon_name}#{suffix}"
      end

      sig { returns(T::Boolean) }
      def current_page?
        current_page_name == page_name
      end
    end
  end
end
